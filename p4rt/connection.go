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
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// Cache of address to gRPC client
var grpcClients = make(map[string]*grpc.ClientConn)

func MonitorConnection(conn *grpc.ClientConn) {
	state := conn.GetState()
	for {
		fmt.Printf("gRPC state update for %s: %v\n", conn.Target(), state.String())
		if state == connectivity.Shutdown {
			break
		}
		conn.WaitForStateChange(context.Background(), state)
		state = conn.GetState()
	}
}

func GetConnection(host string) (conn *grpc.ClientConn, err error) {
	conn, ok := grpcClients[host]
	if !ok {
		conn, err = grpc.Dial(host, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		grpcClients[host] = conn
		go MonitorConnection(conn)
	}
	return
}