# Elevator System  

## How to Run  

### 1. Automatic Restart 
If you want the program to restart automatically after a crash, use the `restart.sh` script:  

```sh
chmod +x restart.sh  # Run this only once to make the script executable
./restart.sh <id> <port>

```

To kill the program, simply interrupt it (CTRL+C) while it's restarting. Press CTRL+C twice in quick succession to forcefully terminate the program.

### 2. Manual Restart

If you prefer to run the program manually without automatic restarts, use:

go run main.go --id <id> --port <port>

## Important Notes

Each elevator must have a unique ID (0, 1, or 2).

<port> refers to the elevator's communication port.