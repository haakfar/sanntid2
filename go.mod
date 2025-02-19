module Main

go 1.18

replace Network-go => ./network

replace Driver-go => ./driver

replace Config => ./config

replace Elevator => ./elevator

require (
	Config v0.0.0-00010101000000-000000000000
	Driver-go v0.0.0-00010101000000-000000000000
	Elevator v0.0.0-00010101000000-000000000000
	Network-go v0.0.0-00010101000000-000000000000
)
