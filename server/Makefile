#
# Makefile for moth server
#
NAME=moth-lite
#--------------------------------------------------------------------------------
usage: 
	@echo "usage: make [build|run|kill|clean] for $(NAME)"
build b:
	go build -ldflags="-s -w" -trimpath -o $(NAME)
	#go build -trimpath -o $(NAME)
	#tinygo build -o $(NAME)

build-clean bc:
	go clean -cache -modcache -testcache

build-platform bp:
	@echo "usage: make (platform) [linux|darwin|windows]"
build-platform-linux bpl:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o $(NAME)
build-platform-darwin bpd:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o $(NAME)
build-platform-windows bpw:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o $(NAME).exe
	
version v:
	go version -m $(NAME)

run r:
	@echo "> make (run) [server|manager|cast|play]"
ps p:
	ps -ef | grep moth | grep -v grep

net n: 
	@echo "> make (net) [port|listen]"
net-port np:
	lsof -i:8276-8277,8433
net-listen nl:
	netstat -anp tcp | grep LISTEN

kill k:
	pkill $(NAME)

clean:
	rm -f $(NAME) $(NAME).exe
	rm -f moth-current.log
#--------------------------------------------------------------------------------
source-line sl:
	cat *.go | wc -l
#--------------------------------------------------------------------------------
build-run br:
	-@make kill 2> /dev/null
	@make build
	@make run-server

build-run-manager brm:
	-@make kill 2> /dev/null
	@make build
	@make run-server
	@make run-manager

build-run-manager2 brm2:
	-@make kill 2> /dev/null
	@make build
	@make run-server
	@make run-manager-2

build-run-control brl:
	-@make kill 2> /dev/null
	@make build
	@make run-server
	@make run-control

docker-kill dk:
	#docker stop $(NAME) && docker rm $(NAME)
	docker kill $(NAME)
#--------------------------------------------------------------------------------
run-server rs:
	./$(NAME) -rtype=server &

run-manager rm:
	./$(NAME) -rtype=manager
run-manager-2 rm2:
	./$(NAME) -rtype=manager2
run-manager-3 rm3:
	./$(NAME) -rtype=manager3

run-monitor ro:
	./$(NAME) -rtype=monitor
run-monitor-2 ro2:
	./$(NAME) -rtype=monitor2

run-control rl:
	./$(NAME) -rtype=control

run-key rk:
	@echo "> make (run-key) [gen|check]"
run-keygen rkg:
	./$(NAME) -rtype=keygen
run-keychk rkc:
	#MOTH_LICENSE_KEY=ckg148i4io89ff28oj80-49.50.162.98-D282770101010010000365:6b7b2a5b ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=co6vf7ichpl7o5qk88jg-192.168.0.102-D282920100010010004383:a91fddf9 ./$(NAME) -rtype=keychk # date error
	#MOTH_LICENSE_KEY=co6vovqchpl94og9d7vg-192.168.0.102-D282920100010010002922!40ce5398 ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cuauuaiaj3j34cimcilg-5e:fd:EE:8d:B3:44-D282070103000501001826!23391c85 ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cui4n4qaj3j190mmi6qg-112.185.208.70-D282080102001000504748!ea76be8e ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cui9ecqaj3j21m214chg-D0:46:0C:58:FB:48-D382080103001000504748!292792fe ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cuibd22aj3jabqobamdg-C40F767AE65B-D382080104001000504748!6bf2308a ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cuib42iaj3j7sl49807g-B40F767AE65B-D382080104001000504748!ef833def ./$(NAME) -rtype=keychk
	#MOTH_LICENSE_KEY=cukmjiaaj3j6po0pojm0-121.146.2.148-D282080100001000502557!d3cce159 ./$(NAME) -rtype=keychk
	MOTH_LICENSE_KEY=cvdpo5l94t025qlsngog-172.31.14.28-D382920100001001004748!7aad622c ./$(NAME) -rtype=keychk
	#./$(NAME) -rtype=keychk
rmb:
	./$(NAME) -rtype=manager -haddr=cobot.center -portp=8286
	#./$(NAME) -rtype=manager -haddr=cobot.center -ports=8287
rmb2:
	./$(NAME) -rtype=manager2 -haddr=cobot.center -portp=8286
	#./$(NAME) -rtype=manager2 -haddr=cobot.center -ports=8287 -key=teamgrit

run-cast rc:
	../tools/video-ws/cast/video-cast -media=../tools/video-ws/cast/sample.mp4 &

run-play rp:
	../tools/video-ws/play/video-play &
#--------------------------------------------------------------------------------
gopls:
	go install golang.org/x/tools/gopls@latest
git-update gu:
	cd ..; make gu
#--------------------------------------------------------------------------------

