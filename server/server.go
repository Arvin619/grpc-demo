package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	pb "github.com/Arvin619/grpc-demo/proto"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
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
		_, err := srs.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				srs.SendAndClose(&pb.HelloReply{Message: "Done~"})
				return nil
			} else {
				panic(err)
			}
		}
		// log.Println(req.Name)
	}
}

func (gs *GreeterServer) SayRoute(srs pb.Greeter_SayRouteServer) error {
	n := 0
	for {
		_ = srs.Send(&pb.HelloReply{Message: "say.route" + strconv.Itoa(n)})
		_, err := srs.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			} else {
				return err
			}
		}
		n++
		// log.Println(req.Name)
	}
}

func createLogFile() (logFile *os.File, closeFun func()) {
	var err error
	if _, err = os.Stat("./log"); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir("./log", os.ModePerm); err != nil {
			panic(err)
		}
	}
	logFileName := time.Now().Format(time.ANSIC) + ".log"
	logFile, err = os.Create(filepath.Join("./log", logFileName))
	if err != nil {
		panic(err)
	}
	closeFun = func() {
		logFile.Close()
	}
	return
}

func InterceptorLogger(l *log.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			msg = fmt.Sprintf("DEBUG :%v", msg)
		case logging.LevelInfo:
			msg = fmt.Sprintf("INFO :%v", msg)
		case logging.LevelWarn:
			msg = fmt.Sprintf("WARN :%v", msg)
		case logging.LevelError:
			msg = fmt.Sprintf("ERROR :%v", msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
		l.Println(append([]any{"msg", msg}, fields...))
	})
}

func main() {
	file, close := createLogFile()
	defer close()

	logger := InterceptorLogger(log.New(io.MultiWriter(os.Stdout, file), "", log.Ldate|log.Ltime|log.Lshortfile))
	logOpt := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(logger, logOpt...),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(logger, logOpt...),
		),
	)
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
