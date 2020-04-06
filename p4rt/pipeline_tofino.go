// +build tofino

/*
 * Copyright 2020-present Brian O'Connor
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// To include this file, use go build -tags tofino

package p4rt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
)

func LoadDeviceConfig(deviceConfigPath string) (P4DeviceConfig, error) {
	paths := strings.Split(deviceConfigPath, ",")
	if len(paths) != 2 {
		return nil, errors.New("Device Config Path is invalid.\n\n" +
			"For Tofino targets, you must provide a comma separated list with two file paths")
	}

	tofinoBinPath := strings.TrimSpace(paths[0])
	tofinoContextPath := strings.TrimSpace(paths[1])

	fmt.Printf("Tofino Bin: %s\nTofino Context: %s\n", tofinoBinPath, tofinoContextPath)

	if !strings.HasSuffix(tofinoBinPath, ".bin") {
		return nil, errors.New("Device Config Path is invalid.\n" +
			"For Tofino targets, the tofino.bin comes first, and must end in \".bin\"")
	}

	if !strings.HasSuffix(tofinoContextPath, ".json") {
		return nil, errors.New("Device Config Path is invalid.\n" +
			"For Tofino targets, the context.json comes second, and must end in \".json\"")
	}

	pipeconfName := "p4rt-go-gen"

	tofinoBin, err := os.Open(tofinoBinPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %v", tofinoBinPath, err)
	}
	defer tofinoBin.Close()
	tofinoBinInfo, err := tofinoBin.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %v", tofinoBinPath, err)
	}

	tofinoContext, err := os.Open(tofinoContextPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %v", tofinoContextPath, err)
	}
	defer tofinoContext.Close()
	tofinoContextInfo, err := tofinoContext.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %v", tofinoContextPath, err)
	}

	// Allocate the device config buffer
	binLen := len(pipeconfName) + int(tofinoBinInfo.Size()) + int(tofinoContextInfo.Size()) + 12 // 3 * 32bit int
	bin := make([]byte, binLen)

	i := 0
	binary.LittleEndian.PutUint32(bin[i:], uint32(len(pipeconfName)))
	i += 4
	if b := copy(bin[i:], pipeconfName); b != len(pipeconfName) {
		panic("pipeconf name copy failed")
	}
	i += len(pipeconfName)

	binary.LittleEndian.PutUint32(bin[i:], uint32(tofinoBinInfo.Size()))
	i += 4
	if b, err := tofinoBin.Read(bin[i:]); err != nil {
		return nil, fmt.Errorf("read %s: %v", tofinoBinPath, err)
	} else if b != int(tofinoBinInfo.Size()) {
		return nil, errors.New("tofino bin copy failed")
	}
	i += int(tofinoBinInfo.Size())

	binary.LittleEndian.PutUint32(bin[i:], uint32(tofinoContextInfo.Size()))
	i += 4
	if b, err := tofinoContext.Read(bin[i:]); err != nil {
		return nil, fmt.Errorf("read %s: %v", tofinoContextPath, err)
	} else if b != int(tofinoContextInfo.Size()) {
		return nil, errors.New("tofino context copy failed")
	}
	i += int(tofinoContextInfo.Size())

	if i != binLen {
		return nil, errors.New("failed to build device config for tofino")
	}

	return bin, nil
}
