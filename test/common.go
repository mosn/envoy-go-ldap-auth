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

package test

import (
	"fmt"
	"os"
	"os/exec"
)

func startEnvoyBind(host string, port int, baseDn, attribute string) {
	startEnvoy(host, port, baseDn, attribute, "", "", "", false)
}

func startEnvoySearch(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string) {
	startEnvoy(host, port, baseDn, attribute, bindDn, bindPassword, filter, false)
}

func startEnvoyTLS(host string, port int, baseDn, attribute string) {
	startEnvoy(host, port, baseDn, attribute, "", "", "", true)
}

func startEnvoy(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string, tls bool) {
	generateEnvoyConfig(host, port, baseDn, attribute, bindDn, bindPassword, filter, tls)
	var err error
	if tls {
		err = exec.Command("bash", "-c", `sed -i "s/host: localhost/host: $(ifconfig eth0 | awk '/inet / {print $2}')/" envoy.yaml`).Run()
		if err != nil {
			panic(fmt.Sprintf("failed to update envoy.yaml: %v", err))
		}
		err = exec.Command("bash", "-c", `awk 'FNR==NR{a=a$0"\\n";next} /rootCA: # ""/{sub(/rootCA: # ""/, "rootCA: \""a"\"")} 1' glauth.crt envoy.yaml > envoy.yaml.tmp && mv envoy.yaml.tmp envoy.yaml`).Run()
		if err != nil {
			panic(fmt.Sprintf("failed to update envoy.yaml: %v", err))
		}
	}
	cmd := exec.Command("cat", "envoy.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to cat envoy.yaml: %v", err))
	}

	err = exec.Command("bash", "-c", "envoy -c envoy.yaml &").Run()
	if err != nil {
		panic(fmt.Sprintf("failed to start envoy: %v", err))
	}
}

func generateEnvoyConfig(host string, port int, baseDn, attribute, bindDn, bindPassword, filter string, tls bool) {
	config := fmt.Sprintf(`
static_resources:

  listeners:
    - name: listener_0
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 10000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                http_filters:
                  - name: envoy.filters.http.golang
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
                      library_id: example
                      library_path: /etc/envoy/libgolang.so
                      plugin_name: envoy-go-ldap-auth
                      plugin_config:
                        "@type": type.googleapis.com/xds.type.v3.TypedStruct
                        value:
                          # required
                          host: %s
                          port: %d
                          baseDn: %s
                          attribute: %s
                          # optional
                          # be used in search mode
                          bindDn: %s # cn=admin,dc=example,dc=com
                          bindPassword: %s # mypassword
                          # if the filter is set, the filter application will run in search mode.
                          filter: %s # (&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%%s))
                          timeout: 60 # unit is second.
                          tls: %t # false
                          startTLS: # false
                          insecureSkipVerify: # false
                          rootCA: # ""

                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: ["*"]
                      routes:
                        - match:
                            prefix: "/"
                          route:
                            host_rewrite_literal: mosn.io
                            cluster: service_mosn_io

  clusters:
    - name: service_mosn_io
      type: LOGICAL_DNS
      # Comment out the following line to test on v6 networks
      dns_lookup_family: V4_ONLY
      load_assignment:
        cluster_name: service_mosn_io
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: mosn.io
                      port_value: 443
      transport_socket:
        name: envoy.transport_sockets.tls
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
          sni: mosn.io
`, host, port, baseDn, attribute, bindDn, bindPassword, filter, tls)

	// Write the configuration to the specified file
	err := os.WriteFile("envoy.yaml", []byte(config), 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to write Envoy configuration to file: %v", err))
	}
}
