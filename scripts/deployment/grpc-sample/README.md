### Go gRPC Application to test proxy

This directory consists of a sample gRPC application, where the server is started as a kubernetes service and the client is run on local machine.

#### Step 1: Start server

The image is already pushed to `divyanshus/greeter-server`. If one wants to push an image themselves, the Dockerfile is present in `server/`.

To start the server, simply apply the manifests yaml:

```bash
$ kubectl apply -f kubernetes-manifests.yaml
```

This should start the server, that we can see as follows:

```bash
$ kubectl get svc
NAME             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
go-grpc-server   ClusterIP   10.100.84.205   <none>        80/TCP    5s
kubernetes       ClusterIP   10.96.0.1       <none>        443/TCP   4d5h

$ kubectl get pods
NAME                              READY   STATUS    RESTARTS   AGE
go-grpc-server-8554c7d864-rj25s   1/1     Running   0          7s
```

The `CLUSTER-IP` can then be used by the client.

#### Step 2: Start the client application

On the local host:

```bash
$ cd client
$ go run main.go --address <cluster-ip> --name <greeting-name>
```
