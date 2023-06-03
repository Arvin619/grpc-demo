package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"strconv"
	"time"

	pb "github.com/Arvin619/grpc-demo/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var port string

func init() {
	flag.StringVar(&port, "p", "9000", "start port number")
	flag.Parse()
}

func main() {
	conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)
	funArr := []func(pb.GreeterClient) error{
		SayHello,
		SayList,
		SayRecord,
		SayRoute,
	}

	for _, f := range funArr {
		err := f(client)
		if err != nil {
			panic(err)
		}
		log.Println("=================================")
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
}

func SayHello(client pb.GreeterClient) error {
	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "Arvin"})
	if err != nil {
		return err
	}
	log.Println(resp.Message)
	return nil
}

func SayList(client pb.GreeterClient) error {
	stream, err := client.SayList(context.Background(), &pb.HelloRequest{Name: "Arvin"})
	if err != nil {
		return err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			} else {
				return err
			}
		}
		log.Println(resp.Message)
	}
}

func SayRecord(client pb.GreeterClient) error {
	stream, err := client.SayRecord(context.Background())
	if err != nil {
		return err
	}
	for i := 0; i <= 6; i++ {
		_ = stream.Send(&pb.HelloRequest{Name: "Arvin" + strconv.Itoa(i)})
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	log.Println(resp.Message)
	return nil
}

func SayRoute(client pb.GreeterClient) error {
	stream, err := client.SayRoute(context.Background())
	if err != nil {
		return err
	}
	for i := 0; i <= 6; i++ {
		_ = stream.Send(&pb.HelloRequest{Name: "Arvin" + strconv.Itoa(i)})
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			} else {
				return err
			}
		}
		log.Println(resp.Message)
	}

	return stream.CloseSend()
}
