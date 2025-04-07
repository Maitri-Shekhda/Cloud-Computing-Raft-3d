
```markdown
**Cloud Computing Raft 3D**

This project implements a distributed system using the Raft consensus algorithm. The system is developed in Go and aims to provide a reliable and fault-tolerant computing environment.

**Features**

- **Distributed Consensus**: Implements the Raft algorithm to achieve consensus among multiple nodes.
- **Fault Tolerance**: Ensures the system can recover from node failures.
- **Scalability**: Designed to scale with the addition of new nodes.

**Installation**

Clone the repository:

```sh
git clone https://github.com/kush9405/Cloud-computing-Raft-3d.git
cd Cloud-computing-Raft-3d
```

Build the project:

```sh
go build
```

**Usage**

Start the Raft cluster on port 8080:

```sh
./Cloud-computing-Raft-3d
```

**Adding a Printer**

You can add a printer using the following `curl` command:

```sh
curl -X POST -H "Content-Type: application/json" -d '{"company":"Creality","model":"Ender 3"}' http://localhost:8080/api/v1/printers
```

**Listing Printers**

You can list all printers using the following `curl` command:

```sh
curl -X GET http://localhost:8080/api/v1/printers
```







```sh
go mod tidy
```
```sh
go build
```

**run first node**
```sh
go run main.go -id node1 -http 127.0.0.1:8001 -raft 127.0.0.1:9001 -data ./data -bootstrap
```

**run second node**
```sh
go run main.go -id node2 -http 127.0.0.1:8002 -raft 127.0.0.1:9002 -data ./data -join 127.0.0.1:8001
```

**run third node**
```sh
go run main.go -id node3 -http 127.0.0.1:8003 -raft 127.0.0.1:9003 -data ./data -join 127.0.0.1:8001
```
**add printer**
```sh
curl -X POST http://localhost:8001/printers -H "Content-Type: application/json" -d '
{
  "id": "printer1",
  "name": "My 3D Printer",
  "model": "Ender 3",
  "status": "idle",
  "temperature": 25,
  "material": "PLA"
}
'
```
**list of printers**
```sh
curl http://localhost:8001/printers
```
**get a specific printer**
```sh
curl http://localhost:8001/printers/printer1
```
**metrics of nodes**
```sh
curl http://localhost:8001/metrics
```
**Contributing**

Contributions are welcome! Please fork the repository and submit a pull request.

