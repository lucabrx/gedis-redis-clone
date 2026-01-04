package main

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lucabrx/gedis/resp"
)

var Handlers = map[string]func([]resp.Value, *resp.Writer) resp.Value{
	"PING":      ping,
	"ECHO":      echo,
	"SET":       set,
	"GET":       get,
	"DEL":       del,
	"EXISTS":    exists,
	"TTL":       ttl,
	"SUBSCRIBE": subscribe,
	"PUBLISH":   publish,
	"COMMAND":   command,
	"CLIENT":    client,
	"INFO":      info,
	"SELECT":    selectDB,
}

var PubSub = struct {
	mu   sync.RWMutex
	Subs map[string][]*resp.Writer
}{
	Subs: make(map[string][]*resp.Writer),
}

type StorageEntry struct {
	Value     string
	ExpiresAt *time.Time
}

var kv = map[string]StorageEntry{}
var kvMu = sync.RWMutex{}

func command(args []resp.Value, w *resp.Writer) resp.Value {
	return resp.Value{Type: "array", Array: []resp.Value{}}
}

func client(args []resp.Value, w *resp.Writer) resp.Value {
	return resp.Value{Type: "string", Str: "OK"}
}

func info(args []resp.Value, w *resp.Writer) resp.Value {
	return resp.Value{Type: "bulk", Bulk: "role:master"}
}

func selectDB(args []resp.Value, w *resp.Writer) resp.Value { // we only use DB 0, placeholder method
	return resp.Value{Type: "string", Str: "OK"}
}

func ping(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) == 0 {
		return resp.Value{Type: "string", Str: "PONG"}
	}
	return resp.Value{Type: "string", Str: args[0].Bulk}
}

func echo(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) != 1 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'echo' command"}
	}
	return resp.Value{Type: "bulk", Bulk: args[0].Bulk}
}

func set(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) < 2 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'set' command"}
	}

	key := args[0].Bulk
	value := args[1].Bulk
	var expiresAt *time.Time

	for i := 2; i < len(args); i++ {
		arg := strings.ToUpper(args[i].Bulk)
		switch arg {
		case "EX":
			if i+1 >= len(args) {
				return resp.Value{Type: "error", Str: "ERR syntax error"}
			}
			seconds, err := strconv.Atoi(args[i+1].Bulk)
			if err != nil {
				return resp.Value{Type: "error", Str: "ERR value is not an integer or out of range"}
			}
			t := time.Now().Add(time.Duration(seconds) * time.Second)
			expiresAt = &t
			i++
		case "PX":
			if i+1 >= len(args) {
				return resp.Value{Type: "error", Str: "ERR syntax error"}
			}
			millis, err := strconv.Atoi(args[i+1].Bulk)
			if err != nil {
				return resp.Value{Type: "error", Str: "ERR value is not an integer or out of range"}
			}
			t := time.Now().Add(time.Duration(millis) * time.Millisecond)
			expiresAt = &t
			i++
		}
	}

	kvMu.Lock()
	kv[key] = StorageEntry{Value: value, ExpiresAt: expiresAt}
	kvMu.Unlock()

	return resp.Value{Type: "string", Str: "OK"}
}

func get(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) != 1 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'get' command"}
	}

	key := args[0].Bulk

	kvMu.Lock()
	defer kvMu.Unlock()

	entry, ok := kv[key]
	if !ok {
		return resp.Value{Type: "null"}
	}

	if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
		delete(kv, key)
		return resp.Value{Type: "null"}
	}

	return resp.Value{Type: "bulk", Bulk: entry.Value}
}

func del(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) == 0 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'del' command"}
	}

	deletedCount := 0
	kvMu.Lock()
	defer kvMu.Unlock()

	for _, arg := range args {
		key := arg.Bulk
		if _, ok := kv[key]; ok {
			delete(kv, key)
			deletedCount++
		}
	}

	return resp.Value{Type: "integer", Num: deletedCount}
}

func exists(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) == 0 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'exists' command"}
	}

	count := 0
	kvMu.Lock()
	defer kvMu.Unlock()

	for _, arg := range args {
		key := arg.Bulk
		entry, ok := kv[key]
		if ok {
			if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
				delete(kv, key)
			} else {
				count++
			}
		}
	}

	return resp.Value{Type: "integer", Num: count}
}

func ttl(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) != 1 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'ttl' command"}
	}

	key := args[0].Bulk

	kvMu.Lock()
	defer kvMu.Unlock()

	entry, ok := kv[key]
	if !ok {
		return resp.Value{Type: "integer", Num: -2}
	}

	if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
		delete(kv, key)
		return resp.Value{Type: "integer", Num: -2}
	}

	if entry.ExpiresAt == nil {
		return resp.Value{Type: "integer", Num: -1}
	}

	duration := time.Until(*entry.ExpiresAt)
	return resp.Value{Type: "integer", Num: int(duration.Seconds())}
}

func subscribe(args []resp.Value, w *resp.Writer) resp.Value {
	PubSub.mu.Lock()
	defer PubSub.mu.Unlock()

	for i, arg := range args {
		channel := arg.Bulk
		PubSub.Subs[channel] = append(PubSub.Subs[channel], w)

		response := resp.Value{
			Type: "array",
			Array: []resp.Value{
				{Type: "bulk", Bulk: "subscribe"},
				{Type: "bulk", Bulk: channel},
				{Type: "integer", Num: i + 1},
			},
		}
		w.Write(response)
	}

	return resp.Value{Type: "ignore"}
}

func publish(args []resp.Value, w *resp.Writer) resp.Value {
	if len(args) != 2 {
		return resp.Value{Type: "error", Str: "ERR wrong number of arguments for 'publish' command"}
	}

	channel := args[0].Bulk
	message := args[1].Bulk

	PubSub.mu.RLock()
	subscribers, ok := PubSub.Subs[channel]
	PubSub.mu.RUnlock()

	if !ok {
		return resp.Value{Type: "integer", Num: 0}
	}

	count := 0
	for _, sub := range subscribers {
		go func(writer *resp.Writer) {
			writer.Write(resp.Value{
				Type: "array",
				Array: []resp.Value{
					{Type: "bulk", Bulk: "message"},
					{Type: "bulk", Bulk: channel},
					{Type: "bulk", Bulk: message},
				},
			})
		}(sub)
		count++
	}

	return resp.Value{Type: "integer", Num: count}
}

func StartExpirationJob() {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			kvMu.Lock()
			sampleCount := 20
			for k, v := range kv {
				if v.ExpiresAt != nil {
					if v.ExpiresAt.Before(time.Now()) {
						delete(kv, k)
					}
				}
				sampleCount--
				if sampleCount <= 0 {
					break
				}
			}
			kvMu.Unlock()
		}
	}()
}
