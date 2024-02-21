package queue_test

import (
	"encoding/json"
	"testing"

	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/stretchr/testify/require"
)

func setupQueue(t *testing.T) (queue.UploadQueue, *require.Assertions) {
	require := require.New(t)

	q := queue.New()

	data, err := json.Marshal(q)
	require.NoError(err)

	require.Equal(`{"Queue":[],"Locked":false}`, string(data))

	return q, require
}

func setupUpload(count int) (upload []*types.Upload) {
	for i := 0; i < count; i++ {
		msg := storagetypes.NewMsgInitProvider(
			"test-address",
			"localhost:3333",
			"1000",
			"test-key",
		)
		upload = append(upload, &types.Upload{Message: msg})
	}

	return
}

func TestAppend(t *testing.T) {
	q, require := setupQueue(t)

	msg := storagetypes.NewMsgInitProvider(
		"test-address",
		"localhost:3333",
		"1000",
		"test-key",
	)

	u := types.Upload{
		Message: msg,
	}

	q.Append(&u)

	data, err := json.Marshal(q)
	require.NoError(err)

	stringQueue := `{"Queue":[{"message":{"creator":"test-address","ip":"localhost:3333","keybase":"test-key","totalspace":"1000"},"callback":null,"error":null,"response":null}],"Locked":false}`
	failStringQueue := `{"Queue":[{"message":null,"callback":null,"error":null,"response":null}],"Locked":false}`

	require.Equal(stringQueue, string(data))
	require.NotEqual(failStringQueue, string(data))

	q.Append(&u)

	data, err = json.Marshal(q)
	require.NoError(err)

	require.Equal(stringQueue, string(data))
}

func TestPrepareMessage(t *testing.T) {
	cases := map[string]struct {
		uq         queue.UploadQueue
		maxMsgSize int
		resultSize int
	}{
		"empty_queue": {
			uq: queue.UploadQueue{
				Locked: true,
			},
			maxMsgSize: 10,
			resultSize: 0,
		},
		"queue_exceed_max": {
			uq: queue.UploadQueue{
				Locked: true,
				Queue:  setupUpload(10),
			},
			maxMsgSize: 1,
			resultSize: 0,
		},
		"queue_msg_length": {
			uq: queue.UploadQueue{
				Locked: true,
				Queue:  setupUpload(1),
			},
			maxMsgSize: 500,
			resultSize: len(setupUpload(1)[0].Message.String()),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			msgs := c.uq.PrepareMessage(c.maxMsgSize)
			var msgSize int
			for _, m := range msgs {
				msgSize += len(m.String())
			}

			if c.resultSize != msgSize {
				t.Log("Expected size: ", c.resultSize, " Result size: ", msgSize)
				t.Fail()
			}
		})
	}
}
