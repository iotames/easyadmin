module github.com/iotames/easyadmin

go 1.26.0

require (
	github.com/iotames/easyconf v1.2.6
	github.com/iotames/easydb v0.7.0
	github.com/iotames/easyserver v1.8.0
	golang.org/x/crypto v0.59.0
)

require (
	github.com/iotames/miniutils v1.0.11 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)

replace (
	github.com/iotames/easyconf => ./lib/easyconf
	github.com/iotames/easydb => ./lib/easydb
	github.com/iotames/easyserver => ./lib/easyserver
)
