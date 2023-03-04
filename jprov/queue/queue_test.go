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
