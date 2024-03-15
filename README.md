# Sanntid

> A peer to peer solution for the TTK4145 elevator project.

## Execute code
The program is excecuted by the following command
```sh
go run main.go -id=el1 -port=15237
```
The `-port` flag is optinal and will default to `15657` if it is left out.
The program must be started with the same id for it it be alle to recover its ordrs from the other elevators from the
network.

# Authors
- Adin Beslagic 
- Håvard Syslak 
- Erlend Rolfsnes
