module github.com/iotames/easyadmin

go 1.26.0

require (
	github.com/iotames/easyconf v1.2.6
	github.com/iotames/easydb v0.7.0
	github.com/iotames/easyserver v1.8.0
	golang.org/x/crypto v0.59.0
	modernc.org/sqlite v1.34.5
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/iotames/miniutils v1.0.11 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
)

replace (
	github.com/iotames/easyconf => ./lib/easyconf
	github.com/iotames/easydb => ./lib/easydb
	github.com/iotames/easyserver => ./lib/easyserver
)
