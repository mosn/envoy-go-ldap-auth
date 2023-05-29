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

package example

import (
	"context"
	"github.com/allegro/bigcache/v3"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"google.golang.org/protobuf/types/known/anypb"
	"time"
)

func init() {

}

type config struct {
	host      string
	port      uint64
	baseDN    string
	attribute string
	bindDN    string
	password  string
	filter    string
	cacheTTL  int32
	timeout   int32
	cache     *bigcache.BigCache
}

type Parser struct {
}

func (p *Parser) Parse(any *anypb.Any) (interface{}, error) {
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
	if cacheTTL, ok := m["cache_ttl"].(float64); ok {
		conf.cacheTTL = int32(cacheTTL)
	}
	// default is 0, which means no cache
	var err error
	if conf.cacheTTL > 0 {
		conf.cache, err = bigcache.New(
			context.Background(),
			bigcache.DefaultConfig(
				time.Duration(conf.cacheTTL)*time.Second,
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if timeout, ok := m["timeout"].(float64); ok {
		conf.timeout = int32(timeout)
	}
	if conf.timeout == 0 {
		conf.timeout = 60
	}
	return conf, nil
}

func (p *Parser) Merge(parent interface{}, child interface{}) interface{} {
	panic("TODO")
}

func ConfigFactory(c interface{}) api.StreamFilterFactory {
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
