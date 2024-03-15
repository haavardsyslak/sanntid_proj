module sanntid

go 1.21

replace (
	Driver-go => ./localelevator/driver-go/
	Network-go => ./Network-go
)

require Network-go v0.0.0-00010101000000-000000000000

require Driver-go v0.0.0-00010101000000-000000000000 // indirect
