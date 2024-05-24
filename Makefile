CLIENT := client
SERVER := server

CERT_HOST := localhost
CERT_PATH := config/certs
CERT_TYPE := rsa:4096

PROTO_PATH := api/proto

create-certs:
	mkdir -p $(CERT_PATH)

	openssl \
		req \
		-newkey $(CERT_TYPE) \
		-new \
		-nodes \
		-x509 \
		-out $(CERT_PATH)/$(CLIENT)-cert.pem \
		-keyout $(CERT_PATH)/$(CLIENT)-key.pem \
		-addext "subjectAltName = DNS:$(CERT_HOST)" \
		-subj "/C=US/ST=California/L=Somewhere/O=My Organization/OU=My Unit/CN=$(CERT_HOST)"

	openssl \
		req \
		-newkey $(CERT_TYPE) \
		-new \
		-nodes \
		-x509 \
		-out $(CERT_PATH)/$(SERVER)-cert.pem \
		-keyout $(CERT_PATH)/$(SERVER)-key.pem \
		-addext "subjectAltName = DNS:$(CERT_HOST)" \
		-subj "/C=US/ST=California/L=Somewhere/O=My Organization/OU=My Unit/CN=$(CERT_HOST)"

build-pb:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_PATH)/*/*.proto
