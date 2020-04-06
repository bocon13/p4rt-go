// +build !bmv2,!tofino

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

// This file is included in no build tag specifies the target platform

package p4rt

import (
	"errors"
)

func LoadDeviceConfig(deviceConfigPath string) (P4DeviceConfig, error) {
	return nil, errors.New("No target type specified at build time. " +
		"You need to rebuild with \"-tags\"")
}
