package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bolg-developers/chat/server/protobuf"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/xid"
	"io"
	"log"
)

type ChatService struct {
	rooms   map[string]*protobuf.Persons                   // 各roomの情報をインメモリにもつことにする
	streams map[string][]protobuf.ChatService_StreamServer // ルームごとにbroadcastするのに必要
}

func NewChatService() *ChatService {
	return &ChatService{
		rooms: map[string]*protobuf.Persons{
			"drug-room": {
				Persons: []*protobuf.Person{
					{
						Name: "ピエール瀧",
						Sex:  "男性",
						Age:  "51",
					},
					{
						Name: "のりピー",
						Sex:  "女性",
						Age:  "48",
					},
					{
						Name: "清原",
						Sex:  "男性",
						Age:  "51",
					},
				},
			},
		},
		streams: make(map[string][]protobuf.ChatService_StreamServer, 0),
	}
}

func (s ChatService) CreateRoom(context.Context, *empty.Empty) (*protobuf.Room, error) {
	id := xid.New().String()
	s.rooms[id] = &protobuf.Persons{}

	// とりあえず10個に制限
	if len(s.rooms) > 10 {
		return nil, errors.New("作成できる部屋数の上限を超えています")
	}

	log.Println("call CreateRoom:", id)

	return &protobuf.Room{Id: id}, nil
}

func (s ChatService) GetRoomAll(context.Context, *empty.Empty) (*protobuf.Rooms, error) {
	ret := &protobuf.Rooms{}
	for id := range s.rooms {
		ret.Rooms = append(ret.Rooms, &protobuf.Room{Id: id})
	}

	log.Println(fmt.Sprintf("call GetRoomAll: %v", ret))

	return ret, nil
}

func (s ChatService) Stream(stream protobuf.ChatService_StreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch e := req.Event.(type) {
		case *protobuf.StreamRequest_JoinRoom_:
			if _, ok := s.rooms[e.JoinRoom.RoomId]; !ok {
				return errors.New("指定されたIDをもつルームは存在しません")
			}

			for _, p := range s.rooms[e.JoinRoom.RoomId].Persons {
				if p.Name == e.JoinRoom.Person.Name {
					return errors.New("指定された名前は既に使用されています")
				}
			}
			s.rooms[e.JoinRoom.RoomId].Persons = append(s.rooms[e.JoinRoom.RoomId].Persons, e.JoinRoom.Person)

			s.streams[e.JoinRoom.RoomId] = append(s.streams[e.JoinRoom.RoomId], stream)

			log.Print("persons:", s.rooms[e.JoinRoom.RoomId].Persons)

			s.broadcast(e.JoinRoom.RoomId, &protobuf.StreamResponse{
				Event: &protobuf.StreamResponse_JoinRoom_{
					JoinRoom: &protobuf.StreamResponse_JoinRoom{
						Person: e.JoinRoom.Person,
					},
				},
			})

		case *protobuf.StreamRequest_LeaveRoom_:
			if _, ok := s.rooms[e.LeaveRoom.RoomId]; !ok {
				return errors.New("指定されたIDをもつルームは存在しません")
			}

			// 該当者を削除
			newPersons := make([]*protobuf.Person, 0)
			for _, p := range s.rooms[e.LeaveRoom.RoomId].Persons {
				if p.Name != e.LeaveRoom.PersonName {
					newPersons = append(newPersons, p)
				}
			}
			if len(newPersons) == len(s.rooms[e.LeaveRoom.RoomId].Persons) {
				return errors.New("指定された名前をもつ人物は存在しません")
			}
			s.rooms[e.LeaveRoom.RoomId].Persons = newPersons

			// 該当者のstreamを削除
			newStreams := make([]protobuf.ChatService_StreamServer, 0)
			for _, strm := range s.streams[e.LeaveRoom.RoomId] {
				if strm != stream {
					newStreams = append(newStreams, strm)
				}
			}
			if len(newStreams) == len(s.streams[e.LeaveRoom.RoomId]) {
				log.Printf("room_id=%s: ストリームの登録解除に失敗しました", e.LeaveRoom.RoomId)
				return errors.New("ストリームの登録解除に失敗しました")
			}

			s.broadcast(e.LeaveRoom.RoomId, &protobuf.StreamResponse{
				Event: &protobuf.StreamResponse_LeaveRoom_{
					LeaveRoom: &protobuf.StreamResponse_LeaveRoom{
						PersonName: e.LeaveRoom.PersonName,
					},
				},
			})

		case *protobuf.StreamRequest_SendMessage_:
			if _, ok := s.rooms[e.SendMessage.RoomId]; !ok {
				return errors.New("指定されたIDをもつルームは存在しません")
			}

			var exist bool
			for _, p := range s.rooms[e.SendMessage.RoomId].Persons {
				if p.Name == e.SendMessage.PersonName {
					exist = true
				}
			}
			if !exist {
				return errors.New("指定された名前をもつ人物は存在しません")
			}

			s.broadcast(e.SendMessage.RoomId, &protobuf.StreamResponse{
				Event: &protobuf.StreamResponse_SendMessage_{
					SendMessage: &protobuf.StreamResponse_SendMessage{
						PersonName: e.SendMessage.PersonName,
						Message: e.SendMessage.Message,
					},
				},
			})

		case *protobuf.StreamRequest_GetRoommateAll_:
			if _, ok := s.rooms[e.GetRoommateAll.RoomId]; !ok {
				return errors.New("指定されたIDをもつルームは存在しません")
			}

			s.broadcast(e.GetRoommateAll.RoomId, &protobuf.StreamResponse{
				Event: &protobuf.StreamResponse_GetRoommateAll_{
					GetRoommateAll: &protobuf.StreamResponse_GetRoommateAll{
						Persons: s.rooms[e.GetRoommateAll.RoomId],
					},
				},
			})
		}
	}
}

func (s ChatService) broadcast(roomId string, res *protobuf.StreamResponse) {
	for _, strm := range s.streams[roomId] {
		if err := strm.Send(res); err != nil {
			log.Printf("ブロードキャストエラー[%s]: %s", roomId, err.Error())
		}
	}
}
