## Placement Package

Given the application graph, generate the optimal placement for a given set of policies.

The implementation uses a SMT formulation to optimize for a particular target for the number of sidecars.

### Tests

To test the SMT formulation:
```bash
go test -v smt_test.go smt.go -args -logtostderr 
```

To test the placement package:
```bash
go test -v placement_test.go placement.go -args -logtostderr
```