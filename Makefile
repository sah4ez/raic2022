build_new_version:
	go build -v -o ./bin/raic2022
	mv ./bin/raic2022_new ./bin/raic2022_old
	mv ./bin/raic2022 ./bin/raic2022_new

build:
	go build -v -o ./bin/raic2022_new

run:
	cd ./bin/
	./aicup22 --config config_empty.json --log-level info

zip:
	zip ./bin/my_stratage.zip ./main.go \
		./go.mod \
		./debugging/ \
		./debugging/debug_command.go \
		./debugging/debug_state.go \
		./debugging/debug_data.go \
		./debugging/color.go \
		./debugging/camera.go \
		./debugging/colored_vertex.go \
		./stream/ \
		./stream/stream.go \
		./debug_interface.go \
		./my_strategy.go \
		./model/ \
		./model/item.go \
		./model/order.go \
		./model/loot.go \
		./model/zone.go \
		./model/game.go \
		./model/constants.go \
		./model/obstacle.go \
		./model/weapon_properties.go \
		./model/unit.go \
		./model/sound.go \
		./model/unit_order.go \
		./model/action_order.go \
		./model/vec2.go \
		./model/action_type.go \
		./model/sound_properties.go \
		./model/projectile.go \
		./model/action.go \
		./model/player.go \
		./codegame/ \
		./codegame/server_message.go \
		./codegame/client_message.go
