.PHONY: build run clean

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

clean:
	docker rm -f envoy-ldap-test