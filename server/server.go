package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	pb "github.com/Arvin619/grpc-demo/proto"
	"google.golang.org/grpc"
)

var port string

func init() {
	flag.StringVar(&port, "p", "9000", "start port number")
	flag.Parse()
}

type GreeterServer struct {
	pb.UnimplementedGreeterServer
}

func (gs *GreeterServer) SayHello(ctx context.Context, pr *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "hello.world"}, nil
}

func (gs *GreeterServer) SayList(pr *pb.HelloRequest, ss pb.Greeter_SayListServer) error {
	for i := 0; i <= 6; i++ {
		_ = ss.Send(&pb.HelloReply{Message: "hello.list" + strconv.Itoa(i)})
	}
	return nil
}

func (gs *GreeterServer) SayRecord(srs pb.Greeter_SayRecordServer) error {
	for {
		req, err := srs.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				srs.SendAndClose(&pb.HelloReply{Message: "Done~"})
				return nil
			} else {
				panic(err)
			}
		}
		log.Println(req.Name)
	}
}

func (gs *GreeterServer) SayRoute(srs pb.Greeter_SayRouteServer) error {
	n := 0
	for {
		_ = srs.Send(&pb.HelloReply{Message: "say.route" + strconv.Itoa(n)})
		req, err := srs.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			} else {
				return err
			}
		}
		n++
		log.Println(req.Name)
	}
}

func main() {
	server := grpc.NewServer()
	pb.RegisterGreeterServer(server, &GreeterServer{})
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		panic(err)
	}
	go func() {
		log.Println("server running " + lis.Addr().String())
		if err := server.Serve(lis); err != nil {
			log.Fatalln(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("server is shuting down...")
	server.GracefulStop()
	log.Println("server is already shutdown")
}
