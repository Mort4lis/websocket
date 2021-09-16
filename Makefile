lint:
	golangci-lint run

test:
	docker run --rm \
		-v "${PWD}:/config" \
  		-v "${PWD}/reports:/reports" \
  		--network=host --name fuzzingclient crossbario/autobahn-testsuite:0.8.2 \
  		wstest -m fuzzingclient -s ./config/fuzzingclient.json && \
  		python3 ./utils/autobahn_res_parser.py --filepath=./reports/index.json --ignore-non-strict