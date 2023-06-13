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
	"crypto/tls"
	"crypto/x509"
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

// parseUsernameAndPassword parses an HTTP Basic Authentication string.
// provide "Basic aGFja2Vyczpkb2dvb2Q=", return ("hackers", "dogood", true).
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

func Connect(conf *config) (*ldap.Conn, error) {
	var rootCA *x509.CertPool

	if conf.rootCA != "" {
		rootCA = x509.NewCertPool()
		rootCA.AppendCertsFromPEM([]byte(conf.rootCA))
	}

	var conn *ldap.Conn = nil
	var err error = nil
	if conf.tls {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: conf.insecureSkipVerify,
			ServerName:         conf.host,
			RootCAs:            rootCA,
		}
		if conf.startTLS {
			conn, err = dial(conf)
			if err == nil {
				err = conn.StartTLS(tlsCfg)
			}
		} else {
			conn, err = dialTLS(conf, tlsCfg)
		}
	} else {
		conn, err = dial(conf)
	}

	if err != nil {
		return nil, err
	}
	return conn, nil
}

func dialTLS(conf *config, tlsCfg *tls.Config) (*ldap.Conn, error) {
	return ldap.DialURL(
		fmt.Sprintf("ldaps://%s:%d", conf.host, conf.port),
		ldap.DialWithTLSDialer(
			tlsCfg,
			&net.Dialer{
				Timeout: time.Duration(conf.timeout) * time.Second,
			}),
	)
}

func dial(conf *config) (*ldap.Conn, error) {
	return ldap.DialURL(
		fmt.Sprintf("ldap://%s:%d", conf.host, conf.port),
		ldap.DialWithDialer(&net.Dialer{
			Timeout: time.Duration(conf.timeout) * time.Second,
		}),
	)
}

// newLdapClient creates a new ldap client.
func newLdapClient(conf *config) (*ldap.Conn, error) {
	client, err := Connect(conf)
	if err != nil {
		return nil, err
	}

	// First bind with a read only user
	err = client.Bind(conf.bindDN, conf.password)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// authLdap authenticates the user against the ldap server.
func (f *filter) authLdap(username, password string) bool {
	if f.config.filter != "" {
		f.callbacks.Log(api.Debug, "running in search mode")
		return f.searchMode(username, password)
	}

	// run with bind mode
	f.callbacks.Log(api.Debug, "running in bind mode")

	client, err := Connect(f.config)
	if err != nil {
		f.callbacks.Log(api.Error, fmt.Sprintf("dial error: %v", err))
		return false
	}

	userDN := fmt.Sprintf("%s=%s,%s", f.config.attribute, username, f.config.baseDN)
	f.callbacks.Log(api.Debug, fmt.Sprintf("Authenticating User: %s", userDN))

	// SimpleBind User and password.
	_, err = client.SimpleBind(&ldap.SimpleBindRequest{
		Username: userDN,
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

	req := ldap.NewSearchRequest(
		f.config.baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf(f.config.filter, username),
		[]string{"dn", "cn"}, nil)

	sr, err := client.Search(req)
	if err != nil {
		f.callbacks.Log(api.Error, fmt.Sprintf("search error: %v", err))
		return
	}

	switch {
	case len(sr.Entries) < 1:
		f.callbacks.Log(api.Debug, "search filter return empty result")
		return
	case len(sr.Entries) > 1:
		f.callbacks.Log(api.Debug, fmt.Sprintf("search filter return multiple entries (%d)", len(sr.Entries)))
		return
	}

	userDN := sr.Entries[0].DN
	f.callbacks.Log(api.Debug, fmt.Sprintf("authenticating user: %s", userDN))

	// Bind User and password.
	err = client.Bind(userDN, password)
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
