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
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	http.RegisterHttpFilterConfigFactory("envoy-go-ldap-auth", configFactory)
	http.RegisterHttpFilterConfigParser(&parser{})
}

type config struct {
	host      string
	port      uint64
	baseDN    string
	attribute string
	bindDN    string
	password  string
	filter    string
	timeout   int32
}

type parser struct {
}

func (p *parser) Parse(any *anypb.Any) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	v := configStruct.Value
	conf := &config{}
	m := v.AsMap()
	if host, ok := m["host"].(string); ok {
		conf.host = host
	}
	if port, ok := m["port"].(float64); ok {
		conf.port = uint64(port)
	}
	if baseDN, ok := m["base_dn"].(string); ok {
		conf.baseDN = baseDN
	}
	if attribute, ok := m["attribute"].(string); ok {
		conf.attribute = attribute
	}
	if bindDN, ok := m["bind_dn"].(string); ok {
		conf.bindDN = bindDN
	}
	if password, ok := m["bind_password"].(string); ok {
		conf.password = password
	}
	if cFilter, ok := m["filter"].(string); ok {
		conf.filter = cFilter
	}

	if timeout, ok := m["timeout"].(float64); ok {
		conf.timeout = int32(timeout)
	}
	if conf.timeout == 0 {
		conf.timeout = 60
	}
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	newConfig := *parentConfig
	if childConfig != nil {
		newConfig = *childConfig
	}
	return &newConfig
}

func configFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type, should not happen")
	}
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}
