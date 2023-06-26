package server_test

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/stretchr/testify/assert"
)

func TestWriteResponse(t *testing.T) {
	cases := map[string]struct {
		fid       string
		cid       string
		hasMsgErr bool
		expErr    bool
	}{
		"no_error_response": {
			fid:       "1",
			cid:       "1",
			hasMsgErr: false,
			expErr:    false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			upload := types.Upload{}

			if c.hasMsgErr {
				upload.Err = errors.New("example error")
			}

			err := server.WriteResponse(rec, upload, c.fid, c.cid)
			assert.NoError(t, err)

			resp := types.UploadResponse{
				CID: c.cid,
				FID: c.fid,
			}

			expResult, err := json.Marshal(resp)
			if err != nil {
				t.Error(err)
			}

			assert.NotNil(t, rec.Body)
			// converted to string for easier reading
			assert.Equal(t, string(expResult)+"\n", rec.Body.String())
		})
	}
}

func TestBuildCid(t *testing.T) {
	cases := map[string]struct {
		address string
		sender  string
		fid     string
		expErr  bool
	}{
		"valid_cid": {
			address: "example_address",
			sender:  "example_sender",
			fid:     "example_fid",
			expErr:  false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cid, err := server.BuildCid(c.address, c.sender, c.fid)

			if c.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			footprint := c.sender + c.address + c.fid

			h := sha256.New()
			_, err = h.Write([]byte(footprint))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			expCid, _ := utils.MakeCid(h.Sum(nil))

			assert.Equal(t, expCid, cid)
		})
	}
}
