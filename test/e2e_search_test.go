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
	"net/http"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	startEnvoySearch("localhost", 3893, "dc=glauth,dc=com", "cn", "cn=serviceuser,ou=svcaccts,dc=glauth,dc=com", "mysecret", "(cn=%s)")
	time.Sleep(5 * time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://localhost:10000/", nil)

	resp1, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp1.StatusCode)
	}

	req.SetBasicAuth("unknown", "dogood")
	resp2, err := http.DefaultClient.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp2.StatusCode)
	}

	req.SetBasicAuth("hackers", "unknown")
	resp3, err := http.DefaultClient.Do(req)
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %v", resp3.StatusCode)
	}

	req.SetBasicAuth("hackers", "dogood")
	resp4, err := http.DefaultClient.Do(req)
	defer resp4.Body.Close()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %v", resp4.StatusCode)
	}

	t.Log("TestSearch passed")
}
