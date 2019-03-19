package main

import (
	"context"
	"github.com/bolg-developers/chat/server/protobuf"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("接続エラー:", err)
	}

	client := protobuf.NewChatServiceClient(conn)

	res, err := client.CreateRoom(context.TODO(), &empty.Empty{})
	if err != nil {
		log.Fatal("CreateRoom関数の呼び出しに失敗:", err)
	}
	log.Printf("room: %v", res)

	res2, err := client.GetRoomAll(context.TODO(), &empty.Empty{})
	if err != nil {
		log.Fatal("GetRoomAll関数の呼び出しに失敗:", err)
	}
	log.Printf("rooms: %v", res2)

	//--------------------------------------------------------------------------
	// bidirectional streaming
	//--------------------------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Stream(ctx)
	if err != nil {
		log.Fatal("can't call JoinRoom:", err)
	}

	rid := os.Args
	log.Print(rid)
	if err := stream.Send(&protobuf.StreamRequest{
		Event: &protobuf.StreamRequest_JoinRoom_{
			JoinRoom: &protobuf.StreamRequest_JoinRoom{
				RoomId: rid[1],
				Person: &protobuf.Person{
					Name: "谷まさる",
					Sex:  "男性",
					Age:  "114514",
				},
			},
		},
	}); err != nil {
		log.Fatal("ルーム入室に失敗:", err)
	}

	if err := stream.Send(&protobuf.StreamRequest{
		Event: &protobuf.StreamRequest_SendMessage_{
			SendMessage: &protobuf.StreamRequest_SendMessage{
				RoomId: rid[1],
				PersonName: "谷まさる",
				Message: "fuck you",
			},
		},
	}); err != nil {
		log.Fatal("メッセージ送信に失敗:", err)
	}

	if err := stream.Send(&protobuf.StreamRequest{
		Event: &protobuf.StreamRequest_GetRoommateAll_{
			GetRoommateAll: &protobuf.StreamRequest_GetRoommateAll{
				RoomId: rid[1],
			},
		},
	}); err != nil {
		log.Fatal("ルームメート取得に失敗:", err)
	}

	//if err := stream.Send(&protobuf.StreamRequest{
	//	Event: &protobuf.StreamRequest_LeaveRoom_{
	//		LeaveRoom: &protobuf.StreamRequest_LeaveRoom{
	//			RoomId: rid[1],
	//			PersonName: "谷まさる",
	//		},
	//	},
	//}); err != nil {
	//	log.Fatal("ルーム退室に失敗:", err)
	//}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Fatalf("受信に失敗: %v", err)
		}
		log.Print("res:", res)
	}
}
