module Elevators

go 1.18

replace Network-go => ./network

replace Driver-go => ./driver

require (
	Driver-go v0.0.0-00010101000000-000000000000
	Network-go v0.0.0-00010101000000-000000000000
)
