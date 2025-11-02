mkdir proto

protoc \
  --proto_path=protobufs/steam \
  --go_out=proto \
  --go_opt=Msteammessages_base.proto=./steam \
  --go_opt=Msteammessages_unified_base.steamclient.proto=./steam \
  --go_opt=Msteammessages_auth.steamclient.proto=./steam \
  --go_opt=Menums.proto=./steam \
  protobufs/steam/steammessages_auth.steamclient.proto \
  protobufs/steam/steammessages_unified_base.steamclient.proto \
  protobufs/steam/steammessages_base.proto \
  protobufs/steam/enums.proto
