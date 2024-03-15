# Sanntid

> A peer to peer solution for the TTK4145 elevator project.

## Execute code
The program is executed by the following command
```sh
go run main.go -id=el1 -port=15237
```
- The `-port` flag is optional and will default to `15657` if it is left out.
- The program, when restarted must be started again with the same id in order to continue serving its orders network.
- Every elevator on the network must have it's own unique id.

# Authors
- Adin Beslagic 
- Håvard Syslak 
- Erlend Rolfsnes
