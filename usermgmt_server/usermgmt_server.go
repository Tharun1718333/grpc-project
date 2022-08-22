package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	pb "example.com/go-usermgmt-grpc/usermgmt"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

func NewUserManagementServer() *UserManagementServer {
	return &UserManagementServer{}
}

type UserManagementServer struct {
	conn *pgx.Conn
	pb.UnimplementedUserManagementServer
}

func (server *UserManagementServer) Run() error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterUserManagementServer(s, server)
	log.Printf("server listening at %v", lis.Addr())
	return s.Serve(lis)

}

func (server *UserManagementServer) CreateNewUser(ctx context.Context, in *pb.NewUser) (*pb.User, error) {
	log.Printf("Recieved %v", in.GetName())
	createSql := `
	create table if not exists users(
		id SERIAL PRIMARY KEY,
		name text,
		age int); 
		`
	_, err := server.conn.Exec(context.Background(), createSql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Table creation failed %v", err)
		os.Exit(1)
	}

	created_user := &pb.User{Name: in.GetName(), Age: in.GetAge()}
	tx, err := server.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("connection failed %v", err)
	}
	_, err = tx.Exec(context.Background(), "insert into users(name , age) values($1 , $2)", created_user.Name, created_user.Age)
	if err != nil {
		log.Fatalf("tx.exec() failed %v", err)
	}
	tx.Commit(context.Background())
	return created_user, nil
}
func (server *UserManagementServer) GetUsers(ctx context.Context, in *pb.GetUserParams) (*pb.UsersList, error) {
	var users_list *pb.UsersList = &pb.UsersList{}
	rows, err := server.conn.Query(context.Background(), "select * from users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		user := pb.User{}
		err := rows.Scan(&user.Id, &user.Name, &user.Age)
		if err != nil {
			return nil, err
		}
		users_list.Users = append(users_list.Users, &user)
	}
	return users_list, nil
}
func main() {
	database_url := "postgres://postgres:postgrespw@localhost:55000/postgres"
	conn, err := pgx.Connect(context.Background(), database_url)
	if err != nil {
		log.Fatalf("unable to establish a connection %v", err)
	}
	defer conn.Close(context.Background())
	var user_mgmt_server *UserManagementServer = NewUserManagementServer()
	user_mgmt_server.conn = conn
	if err := user_mgmt_server.Run(); err != nil {
		log.Fatalf("failure to serve %v", err)
	}
}
