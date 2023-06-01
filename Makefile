.PHONY: build run test-bind-mode test-search-mode test-bind test-search clean

build:
	docker run --rm -v $(PWD):/go/src/go-filter -w /go/src/go-filter \
		-e GOPROXY=https://goproxy.cn \
		golang:1.19 \
		go build -v -o libgolang.so -buildmode=c-shared -buildvcs=false .

run:
	docker rm -f envoy-ldap-test
	docker run --rm -v $(PWD)/example/envoy.yaml:/etc/envoy/envoy.yaml \
		-v $(PWD)/libgolang.so:/etc/envoy/libgolang.so \
		-e "GODEBUG=cgocheck=0" \
		-p 10000:10000 \
		--name envoy-ldap-test \
		envoyproxy/envoy:contrib-v1.26.1 \
		envoy -c /etc/envoy/envoy.yaml

test-bind-mode:
	docker rm -f envoy-ldap-test
	docker run --rm -v $(PWD)/example/envoy.yaml:/etc/envoy/envoy.yaml \
		-v $(PWD)/libgolang.so:/etc/envoy/libgolang.so \
		-e "GODEBUG=cgocheck=0" \
		-p 10000:10000 \
		--name envoy-ldap-test \
		envoyproxy/envoy:contrib-v1.26.1 \
		envoy -c /etc/envoy/envoy.yaml &
	sleep 5
	go test test/e2e_bind_test.go

	docker rm -f envoy-ldap-test

test-search-mode:
	docker rm -f envoy-ldap-test
	docker run --rm -v $(PWD)/example/envoy-search.yaml:/etc/envoy/envoy.yaml \
		-v $(PWD)/libgolang.so:/etc/envoy/libgolang.so \
		-e "GODEBUG=cgocheck=0" \
		-p 10000:10000 \
		--name envoy-ldap-test \
		envoyproxy/envoy:contrib-v1.26.1 \
		envoy -c /etc/envoy/envoy.yaml &
	sleep 5
	go test test/e2e_search_test.go

	docker rm -f envoy-ldap-test

test-bind:
	go test test/e2e_search_test.go

test-search:
	go test test/e2e_search_test.go

clean:
	docker rm -f envoy-ldap-test