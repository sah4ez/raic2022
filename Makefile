build:
	go build -v -o ./bin/raic2022
	mv ./bin/raic2022_new ./bin/raic2022_old
	mv ./bin/raic2022 ./bin/raic2022_new

run:
	./bin/aicup22 --config ./config1.json
