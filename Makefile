lint:
	golangci-lint run

autobahn:
	docker run -it --rm \
		-v "${PWD}:/config" \
  		-v "${PWD}/reports:/reports" \
  		--network=host --name fuzzingclient crossbario/autobahn-testsuite \
  		wstest -m fuzzingclient -s ./config/fuzzingclient.json