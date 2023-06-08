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
	"strconv"
	"syscall"
	"time"

	pb "github.com/Arvin619/grpc-demo/proto"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/natefinch/lumberjack.v2"
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
	if time.Now().Minute()%2 == 0 {
		panic("panic !!!!!!")
	}
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

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	{
		encoderConfig.LevelKey = "level"
		encoderConfig.TimeKey = zapcore.OmitKey
		encoderConfig.MessageKey = "msg"
	}

	return zapcore.NewJSONEncoder(encoderConfig)
}

func createLogFile() *zap.Logger {
	core := zapcore.NewCore(
		getEncoder(),
		zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(
				&lumberjack.Logger{
					Filename:   "./log/grpc-zap.log",
					MaxSize:    100, // megabytes
					MaxBackups: 5,
					MaxAge:     28,   //days
					Compress:   true, // disabled by default
				}),
			zapcore.AddSync(os.Stdout),
		),
		zapcore.DebugLevel,
	)
	logger := zap.New(core)
	return logger
}

func InterceptorLogger(l *zap.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]

			switch v := value.(type) {
			case string:
				f = append(f, zap.String(key.(string), v))
			case int:
				f = append(f, zap.Int(key.(string), v))
			case bool:
				f = append(f, zap.Bool(key.(string), v))
			default:
				f = append(f, zap.Any(key.(string), v))
			}
		}

		logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Info(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

func recoveryHandler(p any) error {
	return status.Errorf(codes.Unknown, "panic triggered: %v", p)
}

func main() {
	logger := createLogFile()
	defer logger.Sync()

	logOpt := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(InterceptorLogger(logger), logOpt...),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(recoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(InterceptorLogger(logger), logOpt...),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(recoveryHandler)),
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
