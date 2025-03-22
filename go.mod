module Main

go 1.18

replace Network-go => ./network

replace Driver-go => ./driver

replace Utils => ./utils

replace Elevator => ./elevator

require Elevator v0.0.0-00010101000000-000000000000

require (
	Driver-go v0.0.0-00010101000000-000000000000 // indirect
	Network-go v0.0.0-00010101000000-000000000000 // indirect
	Utils v0.0.0-00010101000000-000000000000 // indirect
)
