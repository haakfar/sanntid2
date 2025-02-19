package config


const (
	STAND_STILL Direction = iota
	GOING_UP
	GOING_DOWN
)

const (
	SLAVE Role = iota
	BACKUP
	MASTER
)

const N_FLOORS = 4
const N_ELEVATORS = 3

type Direction int
type Role int

type Message struct {
	ID        int                         `json:"id"`
	Role      Role                        `json:"role"`
	Direction Direction                   `json:"direction"`
	Floor     int                         `json:"floor"`
	UpLists    [N_ELEVATORS][N_FLOORS]int  `json:"upLists"`
	DownLists  [N_ELEVATORS][N_FLOORS]int  `json:"downLists"`
	CabLists  [N_ELEVATORS][N_FLOORS]int  `json:"cabLists"`
}