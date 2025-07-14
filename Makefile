#
# Makefile for moth system
#
.PHONY: usage edit build clean git
ORG=cojam
#ARCH=amd64
ARCH=arm64
#ARCH=$(shell arch)
NAME=moth-lite
BASE=$(ORG)/$(NAME)
VERSION=1.1.7
BUILD=$(VERSION).6
DIST=alpine3.20
IMAGE=$(BASE):$(BUILD)-$(DIST)
#IMAGE=$(BASE):$(BUILD)-$(DIST)-$(ARCH)
TOOLS=$(BASE)-tools:$(BUILD)-$(DIST)
GH_IMAGE=ghcr.io/sikang99/$(IMAGE)
#----------------------------------------------------------------------------------
usage:
	@echo "make [edit|build]"
#----------------------------------------------------------------------------------
edit e:
	@echo "make (edit:e) [readme|history]"
edit-readme er:
	vi README.md
edit-history eh:
	vi HISTORY.md
#----------------------------------------------------------------------------------
build b:
	@echo "make (build:b) [full|lite]"
build-full bf:
	cp Dockerfile-full Dockerfile
	make docker-build
build-lite bl:
	cp Dockerfile-lite Dockerfile
	make docker-build
#----------------------------------------------------------------------------------
net-port np:
	lsof -i:8276-8277,8433
#----------------------------------------------------------------------------------
docker d:
	@echo "> make (docker) [build|run|kill|ps] for $(IMAGE)"

docker-buildx-tool-up dbxtu:
	docker buildx build -f Dockerfile.Tools --platform linux/arm64 --tag $(TOOLS)-arm64 .
	docker push $(TOOLS)-arm64
	docker buildx build -f Dockerfile.Tools --platform linux/amd64 --tag $(TOOLS)-amd64 .
	docker push $(TOOLS)-amd64

docker-buildx dbx:
	ARCH=arm64 docker buildx build --platform linux/arm64 --tag $(IMAGE)-arm64 .
	ARCH=amd64 docker buildx build --platform linux/amd64 --tag $(IMAGE)-amd64 .
	docker images $(BASE)

docker-buildx-up dbxu:
	ARCH=arm64 docker buildx build --platform linux/arm64 --tag $(IMAGE)-arm64 .
	docker push $(IMAGE)-arm64
	ARCH=amd64 docker buildx build --platform linux/amd64 --tag $(IMAGE)-amd64 .
	docker push $(IMAGE)-amd64

docker-run dr:
	docker run -d \
		-p 8276-8277:8276-8277/tcp \
		-p 8276-8277:8276-8277/udp \
		-v $(PWD)/server/cert:/moth/cert \
		-v $(PWD)/server/conf:/moth/conf \
		-v $(PWD)/server/html:/moth/html \
		-v $(PWD)/server/data:/moth/data \
		-v $(PWD)/server/log:/moth/log \
		--name $(NAME) $(IMAGE)-arm64

docker-run-base drb:
	docker run -d \
		-p $(PORT):$(PORT) \
		--name $(NAME) $(IMAGE)

docker-exec dx:
	docker exec -it $(NAME) /bin/sh

docker-exec-ip dxp:
	docker exec -it $(NAME) /moth/cmd/externalip

docker-exec-manager dxm:
	docker exec -it $(NAME) moth-server -rtype=manager

docker-exec-keychk dxk:
	docker exec -it $(NAME) moth-server -rtype=keychk

docker-kill dk:
	docker stop $(NAME) && docker rm $(NAME)

docker-test dt:
	docker inspect --format "{{json .State.Health }}" $(IMAGE) | jq

docker-logs dl:
	docker logs $(NAME) -f

docker-ps dp:
	docker ps -a
	-lsof -i TCP:8443
	-lsof -i TCP:8276-8277
	-lsof -i UDP:8276-8277

docker-image di:
	docker images $(BASE)
	docker images $(BASE)-tools

docker-security ds:
	docker scan $(IMAGE)

docker-push du:
	#docker push $(IMAGE)
	docker push $(IMAGE)-arm64
	docker push $(IMAGE)-amd64

docker-build-push dbu:
	docker build -t $(IMAGE) .
	docker push $(IMAGE)

docker-clean dc:
	docker system prune -f
	docker network prune -f
	docker volume prune -f	
#----------------------------------------------------------------------------------
COMPOSE=docker-compose.yml

compose c:
	@echo "> make (compose) [run|kill|ps] with $(COMPOSE)"

compose-run cr:
	docker-compose -f $(COMPOSE) up -d

compose-kill ck:
	docker-compose -f $(COMPOSE) down

compose-exec ce:
	docker-compose -f $(COMPOSE) exec moth moth -rtype=manager

compose-ps cp:
	docker-compose ps

compose-logs cl:
	docker-compose logs -f
#--------------------------------------------------------------------------------
git g:
	@echo "make (git:g) [update|store]"
git-reset gr:
	git reset --soft HEAD~1
git-update gu:
	git add .
	git commit -a -m "$(BUILD),$(USER)"
	git push
git-store gs:
	git config credential.helper store
#----------------------------------------------------------------------------------

