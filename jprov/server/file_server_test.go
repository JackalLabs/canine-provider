package server_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/types"
)

func TestWriteResponse(t *testing.T) {
	cases := map[string]struct{
		fid	string
		cid string
		hasMsgErr bool
		expErr bool
	} {
		"no_error_response": {
			fid: "1",
			cid: "1",
			hasMsgErr: false,
			expErr: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func (t *testing.T){
			rec := httptest.NewRecorder()

			upload := types.Upload{}
			
			if c.hasMsgErr{
				upload.Err = errors.New("example error")
			}

			err := server.WriteResponse(rec, upload, c.fid, c.cid)

			resp := types.UploadResponse {
				CID: c.cid,
				FID: c.fid,
			}
			
			expResult, err := json.Marshal(resp)
			if err != nil {
				t.Error(err)
			}
			
			assert.NotNil(t, rec.Body)
			assert.Equal(t, string(expResult)+"\n", rec.Body.String())	
		})
	}
}
