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

package p4rt

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	p4_config_v1 "github.com/p4lang/p4runtime/proto/p4/config/v1"
	p4 "github.com/p4lang/p4runtime/proto/p4/v1"
	"github.com/pkg/errors"
	"io/ioutil"
)

type P4DeviceConfig []byte

func LoadP4Info(p4infoPath string) (p4info p4_config_v1.P4Info, err error) {
	fmt.Printf("P4 Info: %s\n", p4infoPath)

	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		return
	}
	err = proto.UnmarshalText(string(p4infoBytes), &p4info)
	return
}

func BuildPipelineConfig(p4info p4_config_v1.P4Info, deviceConfigPath string) (config p4.ForwardingPipelineConfig, err error) {
	deviceConfig, err := LoadDeviceConfig(deviceConfigPath)
	if err != nil {
		return
	}

	// Compute the cookie as the hash of the device config
	hash := md5.Sum(deviceConfig)
	cookie := binary.LittleEndian.Uint64(hash[:])

	config.P4Info = &p4info
	config.P4DeviceConfig = deviceConfig
	config.Cookie = &p4.ForwardingPipelineConfig_Cookie{cookie}
	return
}

func getPipelineConfig(client p4.P4RuntimeClient, deviceId uint64) (*p4.ForwardingPipelineConfig, error) {
	req := &p4.GetForwardingPipelineConfigRequest{
		DeviceId:     deviceId,
		ResponseType: p4.GetForwardingPipelineConfigRequest_P4INFO_AND_COOKIE,
	}
	res, err := client.GetForwardingPipelineConfig(context.Background(), req)

	//TODO update ErrorDesc to use non-deprecated method
	//if grpc.ErrorDesc(err) == "No forwarding pipeline config set for this device" {
	//	fmt.Println("no forwarding pipeline; need to push one")
	//	return &p4.ForwardingPipelineConfig{}, nil
	//} else
	if err != nil {
		return nil, errors.Wrap(err, "error getting pipeline config")
	}
	return res.GetConfig(), nil
}

func setPipelineConfig(client p4.P4RuntimeClient, deviceId uint64, electionId *p4.Uint128, config *p4.ForwardingPipelineConfig) error {
	req := &p4.SetForwardingPipelineConfigRequest{
		DeviceId: deviceId,
		RoleId:   0, // not used
		ElectionId: electionId,
		Action: p4.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config: config,
	}
	_, err := client.SetForwardingPipelineConfig(context.Background(), req)
	// ignore the response; it is an empty message
	return err
}

func (c *p4rtClient) SetForwardingPipelineConfig(p4InfoPath, deviceConfigPath string) (err error) {
	p4info, err := LoadP4Info(p4InfoPath)
	if err != nil {
		return
	}
	pipeline, err := BuildPipelineConfig(p4info, deviceConfigPath)
	if err != nil {
		return
	}
	err = setPipelineConfig(c.client, c.deviceId, &c.electionId, &pipeline)
	if err != nil {
		return
	}
	return
}

func (c *p4rtClient) GetForwardingPipelineConfig() (*p4.ForwardingPipelineConfig, error) {
	return getPipelineConfig(c.client, c.deviceId)
}

/* FIXME(bocon)

func matches(target, actual *p4.ForwardingPipelineConfig) bool {
	// TODO Tofino doesn't appear to fill in the device config on Get, so assume it matches
	// When it does, we can replace this with proto compare: proto.Equal(target, actual)
	// TODO consider using cookie for comparision
	return proto.Equal(target.P4Info, actual.P4Info)
}

func UpdatePipelineConfig(client p4.P4RuntimeClient, p4Info *p4_config_v1.P4Info,
	config PipelineConfig, deviceId uint64, forcePush bool) (bool, error) {
	configData, err := config.Get()
	if err != nil {
		return false, errors.Wrap(err, "error building target config")
	}
	targetConfig := &p4.ForwardingPipelineConfig{
		P4Info:         p4Info,
		P4DeviceConfig: configData,
	}

	deviceConfig, err := GetPipelineConfigs(client, deviceId)
	if err != nil {
		return false, errors.Wrap(err, "error getting device config")
	}

	if forcePush || !matches(targetConfig, deviceConfig) {
		// Config doesn't match or updated is forced, so re-push...
		err = setPipelineConfig(client, targetConfig)
		if err != nil {
			return true, errors.Wrap(err, "error setting config")
		}
		return true, nil
	}
	return false, nil
}

*/
