// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: High-level file comment.
package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
)

// Manage the SSH Server
type Server struct {
	ServerConfig   ssh.ServerConfig
	Socket         net.Listener
	AuthorizedKeys map[string]bool
}

var keyNames = []string{
	"ssh_host_dsa_key",
	"ssh_host_ecdsa_key",
	"ssh_host_rsa_key",
}

func NewServer() *Server {
	s := &Server{}
	s.AuthorizedKeys = make(map[string]bool)
	s.ServerConfig.PublicKeyCallback = s.VerifyPublicKey
	return s
}

func (s *Server) ListenAndServe(port int16) error {
	sPort := ":" + strconv.Itoa(int(port))
	if sock, err := net.Listen("tcp", sPort); err != nil {
		dbg.Debug("Unable to listen: %v", err)
		return err
	} else {
		dbg.Debug("Listening on %s", sPort)
		s.Socket = sock
	}
	for {
		conn, err := s.Socket.Accept()
		if err != nil {
			dbg.Debug("Unable to accept: %v", err)
			continue
		}
		dbg.Debug("Accepted connection from: %s", conn.RemoteAddr())

		sConn, err := NewServerConn(conn, s)
		if err != nil {
			if err == io.EOF {
				dbg.Debug("Connection closed by remote host.")
				continue
			}
			dbg.Debug("Unable to negotiate SSH: %v", err)
			continue
		}
		dbg.Debug("Authenticated client from: %s", sConn.RemoteAddr())

		go sConn.HandleConn()
	}
}

func (s *Server) AddAuthorizedKeys(keyData []byte) {
	for len(keyData) > 0 {
		newKey, _, _, left, err := ssh.ParseAuthorizedKey(keyData)
		keyData = left
		if err != nil {
			dbg.Debug("Error parsing key: %v", err)
			break
		}
		s.AuthorizedKeys[string(newKey.Marshal())] = true
	}
}

func (s *Server) VerifyPublicKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	keyStr := string(key.Marshal())
	if _, ok := s.AuthorizedKeys[keyStr]; !ok {
		dbg.Debug("Key not found!")
		return nil, fmt.Errorf("No valid key found.")
	}
	return &ssh.Permissions{}, nil
}

func (s *Server) AddHostkey(keyData []byte) {
	key, err := ssh.ParsePrivateKey(keyData)
	if err == nil {
		s.ServerConfig.AddHostKey(key)
	}
}