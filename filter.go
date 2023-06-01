/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/base64"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/go-ldap/ldap/v3"
	"net"
	"strings"
	"time"
)

type filter struct {
	callbacks api.FilterCallbackHandler
	config    *config
}

func parseUsernameAndPassword(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

func dial(config *config) (*ldap.Conn, error) {
	return ldap.DialURL(
		// TODO: support TLS
		fmt.Sprintf("ldap://%s:%d", config.host, config.port),
		ldap.DialWithDialer(&net.Dialer{
			Timeout: time.Duration(config.timeout) * time.Second,
		}),
	)
}

// newLdapClient creates a new ldap client.
func newLdapClient(config *config) (*ldap.Conn, error) {
	client, err := dial(config)
	if err != nil {
		return nil, err
	}

	err = client.Bind(config.bindDN, config.password)
	// First bind with a read only user
	if err != nil {
		return nil, err
	}
	return client, nil
}

// authLdap authenticates the user against the ldap server.
func (f *filter) authLdap(username, password string) bool {
	if f.config.filter != "" {
		return f.searchMode(username, password)
	}

	// run with bind mode
	client, err := dial(f.config)
	if err != nil {
		f.callbacks.Log(api.Error, fmt.Sprintf("dial error: %v", err))
		return false
	}

	_, err = client.SimpleBind(&ldap.SimpleBindRequest{
		Username: fmt.Sprintf("%s=%s,%s", f.config.attribute, username, f.config.baseDN),
		Password: password,
	})
	return err == nil
}

func (f *filter) searchMode(username, password string) (auth bool) {
	client, err := newLdapClient(f.config)
	if err != nil {
		f.callbacks.Log(api.Error, fmt.Sprintf("newLdapClient error: %v", err))
		return
	}
	defer func() {
		if client != nil {
			client.Close()
		}
		err := recover()
		if err != nil {
			auth = false
			return
		}
	}()

	req := ldap.NewSearchRequest(f.config.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(f.config.filter, username),
		[]string{f.config.attribute}, nil)

	sr, err := client.Search(req)
	if err != nil {
		f.callbacks.Log(api.Debug, fmt.Sprintf("search error: %v", err))
		return
	}

	if len(sr.Entries) != 1 {
		f.callbacks.Log(api.Debug, fmt.Sprintf("search not found: %v", err))
		return
	}

	userDn := sr.Entries[0].DN
	err = client.Bind(userDn, password)
	if err != nil {
		f.callbacks.Log(api.Debug, fmt.Sprintf("bind error: %v", err))
		return
	}

	auth = true
	return
}

func (f *filter) verify(header api.RequestHeaderMap) (bool, string) {
	auth, ok := header.Get("authorization")
	if !ok {
		return false, "no Authorization"
	}

	username, password, ok := parseUsernameAndPassword(auth)
	if !ok {
		return false, "invalid Authorization format"
	}
	ok = f.authLdap(username, password)
	if !ok {
		return false, "invalid username or password"
	}
	return true, ""
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	go func() {
		if ok, msg := f.verify(header); !ok {
			// TODO: set the WWW-Authenticate response header
			f.callbacks.SendLocalReply(401, msg, map[string]string{}, 0, "bad-request")
			return
		}
		f.callbacks.Continue(api.Continue)
	}()
	return api.Running
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
}

func main() {
}
