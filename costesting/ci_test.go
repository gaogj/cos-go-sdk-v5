package cos

// Basic imports
import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type CosTestSuite struct {
	suite.Suite
	VariableThatShouldStartAtFive int

	// CI client
	Client *cos.Client

	// Copy source client
	CClient *cos.Client

	Region string

	Bucket string

	Appid string

	// test_object
	TestObject string

	// special_file_name
	SepFileName string
}

func (s *CosTestSuite) SetupSuite() {
	fmt.Println("Set up test")
	// init
	s.TestObject = "test.txt"
	s.SepFileName = "中文" + "→↓←→↖↗↙↘! \"#$%&'()*+,-./0123456789:;<=>@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

	// CI client for test interface
	// URL like this http://test-1253846586.cos.ap-guangzhou.myqcloud.com
	u := "http://cosgosdktest-1251668577.cos.ap-guangzhou.myqcloud.com"

	// Get the region
	iu, _ := url.Parse(u)
	p := strings.Split(iu.Host, ".")
	assert.Equal(s.T(), 5, len(p), "Bucket host is not right")
	s.Region = p[2]

	// Bucket name
	pp := strings.Split(p[0], "-")
	s.Bucket = pp[0]
	s.Appid = pp[1]

	ib := &cos.BaseURL{BucketURL: iu}
	s.Client = cos.NewClient(ib, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("COS_SECRETID"),
			SecretKey: os.Getenv("COS_SECRETKEY"),
		},
	})

	opt := &cos.BucketPutOptions{
		XCosACL: "public-read",
	}
	r, err := s.Client.Bucket.Put(context.Background(), opt)
	if err != nil && r.StatusCode == 409 {
		fmt.Println("BucketAlreadyOwnedByYou")
	} else if err != nil {
		assert.Nil(s.T(), err, "PutBucket Failed")
	}
}

// Begin of api test

// Service API
func (s *CosTestSuite) TestGetService() {
	_, _, err := s.Client.Service.Get(context.Background())
	assert.Nil(s.T(), err, "GetService Failed")
}

// Bucket API
func (s *CosTestSuite) TestPutHeadDeleteBucket() {
	// Notic sometimes the bucket host can not analyis, may has i/o timeout problem
	u := "http://gosdkbuckettest-" + s.Appid + ".cos.ap-beijing-1.myqcloud.com"
	iu, _ := url.Parse(u)
	ib := &cos.BaseURL{BucketURL: iu}
	client := cos.NewClient(ib, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("COS_SECRETID"),
			SecretKey: os.Getenv("COS_SECRETKEY"),
		},
	})
	r, err := client.Bucket.Put(context.Background(), nil)
	if err != nil && r.StatusCode == 409 {
		fmt.Println("BucketAlreadyOwnedByYou")
	} else if err != nil {
		assert.Nil(s.T(), err, "PutBucket Failed")
	}

	if err != nil {
		panic(err)
	}

	time.Sleep(3 * time.Second)

	_, err = client.Bucket.Head(context.Background())
	assert.Nil(s.T(), err, "HeadBucket Failed")
	if err == nil {
		_, err = client.Bucket.Delete(context.Background())
		assert.Nil(s.T(), err, "DeleteBucket Failed")
	}
}

func (s *CosTestSuite) TestPutBucketACLIllegal() {
	opt := &cos.BucketPutACLOptions{
		Header: &cos.ACLHeaderOptions{
			XCosACL: "public-read-writ",
		},
	}
	_, err := s.Client.Bucket.PutACL(context.Background(), opt)
	assert.NotNil(s.T(), err, "PutBucketACL illegal Failed")
}

func (s *CosTestSuite) TestPutGetBucketACLNormal() {
	// with header
	opt := &cos.BucketPutACLOptions{
		Header: &cos.ACLHeaderOptions{
			XCosACL: "private",
		},
	}
	_, err := s.Client.Bucket.PutACL(context.Background(), opt)
	assert.Nil(s.T(), err, "PutBucketACL normal Failed")

	v, _, err := s.Client.Bucket.GetACL(context.Background())
	assert.Nil(s.T(), err, "GetBucketACL normal Failed")
	assert.Equal(s.T(), 1, len(v.AccessControlList), "GetBucketACL normal Failed, must be private")

}

func (s *CosTestSuite) TestGetBucket() {
	opt := &cos.BucketGetOptions{
		Prefix:  "中文",
		MaxKeys: 3,
	}
	_, _, err := s.Client.Bucket.Get(context.Background(), opt)
	assert.Nil(s.T(), err, "GetBucket Failed")
}

func (s *CosTestSuite) TestGetBucketLocation() {
	v, _, err := s.Client.Bucket.GetLocation(context.Background())
	assert.Nil(s.T(), err, "GetLocation Failed")
	assert.Equal(s.T(), s.Region, v.Location, "GetLocation wrong region")
}

func (s *CosTestSuite) TestPutGetDeleteCORS() {
	opt := &cos.BucketPutCORSOptions{
		Rules: []cos.BucketCORSRule{
			{
				AllowedOrigins: []string{"http://www.qq.com"},
				AllowedMethods: []string{"PUT", "GET"},
				AllowedHeaders: []string{"x-cos-meta-test", "x-cos-xx"},
				MaxAgeSeconds:  500,
				ExposeHeaders:  []string{"x-cos-meta-test1"},
			},
		},
	}
	_, err := s.Client.Bucket.PutCORS(context.Background(), opt)
	assert.Nil(s.T(), err, "PutBucketCORS Failed")

	v, _, err := s.Client.Bucket.GetCORS(context.Background())
	assert.Nil(s.T(), err, "GetBucketCORS Failed")
	assert.Equal(s.T(), 1, len(v.Rules), "GetBucketCORS wrong number rules")
}

func (s *CosTestSuite) TestVersionAndReplication() {
	opt := &cos.BucketPutVersionOptions{
		// Enabled or Suspended, the versioning once opened can not close.
		Status: "Enabled",
	}
	_, err := s.Client.Bucket.PutVersioning(context.Background(), opt)
	assert.Nil(s.T(), err, "PutVersioning Failed")
	v, _, err := s.Client.Bucket.GetVersioning(context.Background())
	assert.Nil(s.T(), err, "GetVersioning Failed")
	assert.Equal(s.T(), "Enabled", v.Status, "Get Wrong Version status")

	repOpt := &cos.PutBucketReplicationOptions{
		// qcs::cam::uin/[UIN]:uin/[Subaccount]
		Role: "qcs::cam::uin/2779643970:uin/2779643970",
		Rule: []cos.BucketReplicationRule{
			{
				ID: "1",
				// Enabled or Disabled
				Status: "Enabled",
				Destination: &cos.ReplicationDestination{
					// qcs::cos:[Region]::[Bucketname-Appid]
					Bucket: "qcs::cos:ap-beijing::alanbj-1251668577",
				},
			},
		},
	}

	_, err = s.Client.Bucket.PutBucketReplication(context.Background(), repOpt)
	assert.Nil(s.T(), err, "PutBucketReplication Failed")
	vr, _, err := s.Client.Bucket.GetBucketReplication(context.Background())
	assert.Nil(s.T(), err, "GetBucketReplication Failed")
	for _, r := range vr.Rule {
		assert.Equal(s.T(), "Enabled", r.Status, "Get Wrong Version status")
		assert.Equal(s.T(), "qcs::cos:ap-beijing::alanbj-1251668577", r.Destination.Bucket, "Get Wrong Version status")

	}
	_, err = s.Client.Bucket.DeleteBucketReplication(context.Background())
	assert.Nil(s.T(), err, "DeleteBucketReplication Failed")
}

func (s *CosTestSuite) TestBucketInventory() {
	id := "test1"
	opt := &cos.BucketPutInventoryOptions{
		ID: id,
		// True or False
		IsEnabled:              "True",
		IncludedObjectVersions: "All",
		Filter: &cos.BucketInventoryFilter{
			Prefix: "test",
		},
		OptionalFields: &cos.BucketInventoryOptionalFields{
			BucketInventoryFields: []string{
				"Size", "LastModifiedDate",
			},
		},
		Schedule: &cos.BucketInventorySchedule{
			// Weekly or Daily
			Frequency: "Daily",
		},
		Destination: &cos.BucketInventoryDestination{
			BucketDestination: &cos.BucketInventoryDestinationContent{
				Bucket: "qcs::cos:ap-guangzhou::alangz-1251668577",
				Format: "CSV",
			},
		},
	}
	_, err := s.Client.Bucket.PutBucketInventoryTest(context.Background(), id, opt)
	assert.Nil(s.T(), err, "PutBucketInventory Failed")
	v, _, err := s.Client.Bucket.GetBucketInventoryTest(context.Background(), id)
	assert.Nil(s.T(), err, "GetBucketInventory Failed")
	assert.Equal(s.T(), "test1", v.ID, "Get Wrong inventory id")
	assert.Equal(s.T(), "true", v.IsEnabled, "Get Wrong inventory isenabled")
	assert.Equal(s.T(), "qcs::cos:ap-guangzhou::alangz-1251668577", v.Destination.BucketDestination.Bucket, "Get Wrong inventory isenabled")

	_, err = s.Client.Bucket.DeleteBucketInventoryTest(context.Background(), id)
	assert.Nil(s.T(), err, "DeleteBucketInventory Failed")

}

func (s *CosTestSuite) TestBucketLogging() {
	opt := &cos.BucketPutLoggingOptions{
		LoggingEnabled: &cos.BucketLoggingEnabled{
			// The bucket must same region.
			TargetBucket: "alangz-1251668577",
		},
	}
	_, err := s.Client.Bucket.PutBucketLoggingTest(context.Background(), opt)
	assert.Nil(s.T(), err, "PutBucketLogging Failed")
	v, _, err := s.Client.Bucket.GetBucketLoggingTest(context.Background())
	assert.Nil(s.T(), err, "GetBucketLogging Failed")
	assert.Equal(s.T(), "alangz-1251668577", v.LoggingEnabled.TargetBucket, "Get Wrong Version status")

}

func (s *CosTestSuite) TestPutGetDeleteLifeCycle() {
	lc := &cos.BucketPutLifecycleOptions{
		Rules: []cos.BucketLifecycleRule{
			{
				ID:     "1234",
				Filter: &cos.BucketLifecycleFilter{Prefix: "test"},
				Status: "Enabled",
				Transition: &cos.BucketLifecycleTransition{
					Days:         10,
					StorageClass: "Standard",
				},
			},
		},
	}
	_, err := s.Client.Bucket.PutLifecycle(context.Background(), lc)
	assert.Nil(s.T(), err, "PutBucketLifecycle Failed")
	_, r, err := s.Client.Bucket.GetLifecycle(context.Background())
	// Might cleaned by other case concrrent
	if err != nil && 404 != r.StatusCode {
		assert.Nil(s.T(), err, "GetBucketLifecycle Failed")
	}
	_, err = s.Client.Bucket.DeleteLifecycle(context.Background())
	assert.Nil(s.T(), err, "DeleteBucketLifecycle Failed")
}

func (s *CosTestSuite) TestListMultipartUploads() {
	// Create new upload
	name := "test_multipart" + time.Now().Format(time.RFC3339)
	flag := false
	v, _, err := s.Client.Object.InitiateMultipartUpload(context.Background(), name, nil)
	assert.Nil(s.T(), err, "InitiateMultipartUpload Failed")
	id := v.UploadID

	// List
	r, _, err := s.Client.Bucket.ListMultipartUploads(context.Background(), nil)
	assert.Nil(s.T(), err, "ListMultipartUploads Failed")
	for _, p := range r.Uploads {
		if p.Key == name {
			assert.Equal(s.T(), id, p.UploadID, "ListMultipartUploads wrong uploadid")
			flag = true
		}
	}
	assert.Equal(s.T(), true, flag, "ListMultipartUploads wrong key")

	// Abort
	_, err = s.Client.Object.AbortMultipartUpload(context.Background(), name, id)
	assert.Nil(s.T(), err, "AbortMultipartUpload Failed")
}

// Object API
func (s *CosTestSuite) TestPutHeadGetDeleteObject_10MB() {
	name := "test/objectPut" + time.Now().Format(time.RFC3339)
	b := make([]byte, 1024*1024*10)
	_, err := rand.Read(b)
	content := fmt.Sprintf("%X", b)
	f := strings.NewReader(content)

	_, err = s.Client.Object.Put(context.Background(), name, f, nil)
	assert.Nil(s.T(), err, "PutObject Failed")

	_, err = s.Client.Object.Head(context.Background(), name, nil)
	assert.Nil(s.T(), err, "HeadObject Failed")

	_, err = s.Client.Object.Delete(context.Background(), name)
	assert.Nil(s.T(), err, "DeleteObject Failed")
}

func (s *CosTestSuite) TestPutGetDeleteObjectByFile_10MB() {
	// Create tmp file
	filePath := "tmpfile" + time.Now().Format(time.RFC3339)
	newfile, err := os.Create(filePath)
	assert.Nil(s.T(), err, "create tmp file Failed")
	defer newfile.Close()

	name := "test/objectPutByFile" + time.Now().Format(time.RFC3339)
	b := make([]byte, 1024*1024*10)
	_, err = rand.Read(b)

	newfile.Write(b)
	_, err = s.Client.Object.PutFromFile(context.Background(), name, filePath, nil)
	assert.Nil(s.T(), err, "PutObject Failed")

	// Over write tmp file
	_, err = s.Client.Object.GetToFile(context.Background(), name, filePath, nil)
	assert.Nil(s.T(), err, "HeadObject Failed")

	_, err = s.Client.Object.Delete(context.Background(), name)
	assert.Nil(s.T(), err, "DeleteObject Failed")

	// remove the local tmp file
	err = os.Remove(filePath)
	assert.Nil(s.T(), err, "remove local file Failed")
}

func (s *CosTestSuite) TestPutGetDeleteObjectSpecialName() {
	f := strings.NewReader("test")
	name := s.SepFileName + time.Now().Format(time.RFC3339)
	_, err := s.Client.Object.Put(context.Background(), name, f, nil)
	assert.Nil(s.T(), err, "PutObject Failed")

	resp, err := s.Client.Object.Get(context.Background(), name, nil)
	assert.Nil(s.T(), err, "GetObject Failed")
	defer resp.Body.Close()
	bs, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(s.T(), "test", string(bs), "GetObject failed content wrong")

	_, err = s.Client.Object.Delete(context.Background(), name)
	assert.Nil(s.T(), err, "DeleteObject Failed")
}

func (s *CosTestSuite) TestPutObjectToNonExistBucket() {
	u := "http://gosdknonexistbucket-" + s.Appid + ".cos." + s.Region + ".myqcloud.com"
	iu, _ := url.Parse(u)
	ib := &cos.BaseURL{BucketURL: iu}
	client := cos.NewClient(ib, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("COS_SECRETID"),
			SecretKey: os.Getenv("COS_SECRETKEY"),
		},
	})
	name := "test/objectPut.go"
	f := strings.NewReader("test")
	r, err := client.Object.Put(context.Background(), name, f, nil)
	assert.NotNil(s.T(), err, "PutObject ToNonExistBucket Failed")
	assert.Equal(s.T(), 404, r.StatusCode, "PutObject ToNonExistBucket, not 404")
}

func (s *CosTestSuite) TestPutGetObjectACL() {
	name := "test/objectACL.go" + time.Now().Format(time.RFC3339)
	f := strings.NewReader("test")
	_, err := s.Client.Object.Put(context.Background(), name, f, nil)
	assert.Nil(s.T(), err, "PutObject Failed")

	// Put acl
	opt := &cos.ObjectPutACLOptions{
		Header: &cos.ACLHeaderOptions{
			XCosACL: "public-read",
		},
	}
	_, err = s.Client.Object.PutACL(context.Background(), name, opt)
	assert.Nil(s.T(), err, "PutObjectACL Failed")
	v, _, err := s.Client.Object.GetACL(context.Background(), name)
	assert.Nil(s.T(), err, "GetObjectACL Failed")
	assert.Equal(s.T(), 2, len(v.AccessControlList), "GetLifecycle wrong number rules")

	_, err = s.Client.Object.Delete(context.Background(), name)
	assert.Nil(s.T(), err, "DeleteObject Failed")
}

func (s *CosTestSuite) TestPutObjectRestore() {
	name := "archivetest"
	putOpt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			XCosStorageClass: "ARCHIVE",
		},
	}
	f := strings.NewReader("test")
	_, err := s.Client.Object.Put(context.Background(), name, f, putOpt)
	assert.Nil(s.T(), err, "PutObject Archive faild")
	opt := &cos.ObjectRestoreOptions{
		Days: 2,
		Tier: &cos.CASJobParameters{
			// Standard, Exepdited and Bulk
			Tier: "Expedited",
		},
	}
	resp, _ := s.Client.Object.PostRestore(context.Background(), name, opt)
	retCode := resp.StatusCode
	if retCode != 200 && retCode != 202 && retCode != 409 {
		right := false
		fmt.Println("PutObjectRestore get code is:", retCode)
		assert.Equal(s.T(), true, right, "PutObjectRestore Failed")
	}

}

func (s *CosTestSuite) TestCopyObject() {
	u := "http://gosdkcopytest-" + s.Appid + ".cos.ap-beijing-1.myqcloud.com"
	iu, _ := url.Parse(u)
	ib := &cos.BaseURL{BucketURL: iu}
	c := cos.NewClient(ib, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  os.Getenv("COS_SECRETID"),
			SecretKey: os.Getenv("COS_SECRETKEY"),
		},
	})

	opt := &cos.BucketPutOptions{
		XCosACL: "public-read",
	}

	// Notice in intranet the bucket host sometimes has i/o timeout problem
	r, err := c.Bucket.Put(context.Background(), opt)
	if err != nil && r.StatusCode == 409 {
		fmt.Println("BucketAlreadyOwnedByYou")
	} else if err != nil {
		assert.Nil(s.T(), err, "PutBucket Failed")
	}

	source := "test/objectMove1" + time.Now().Format(time.RFC3339)
	expected := "test"
	f := strings.NewReader(expected)

	r, err = c.Object.Put(context.Background(), source, f, nil)
	assert.Nil(s.T(), err, "PutObject Failed")
	var version_id string
	if r.Header["X-Cos-Version-Id"] != nil {
		version_id = r.Header.Get("X-Cos-Version-Id")
	}

	time.Sleep(3 * time.Second)
	// Copy file
	soruceURL := fmt.Sprintf("%s/%s", iu.Host, source)
	dest := "test/objectMove1" + time.Now().Format(time.RFC3339)
	//opt := &cos.ObjectCopyOptions{}
	if version_id == "" {
		_, _, err = s.Client.Object.Copy(context.Background(), dest, soruceURL, nil)
	} else {
		_, _, err = s.Client.Object.Copy(context.Background(), dest, soruceURL, nil, version_id)
	}
	assert.Nil(s.T(), err, "PutObjectCopy Failed")

	// Check content
	resp, err := s.Client.Object.Get(context.Background(), dest, nil)
	assert.Nil(s.T(), err, "GetObject Failed")
	bs, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	result := string(bs)
	assert.Equal(s.T(), expected, result, "PutObjectCopy Failed, wrong content")
}

func (s *CosTestSuite) TestCreateAbortMultipartUpload() {
	name := "test_multipart" + time.Now().Format(time.RFC3339)
	v, _, err := s.Client.Object.InitiateMultipartUpload(context.Background(), name, nil)
	assert.Nil(s.T(), err, "InitiateMultipartUpload Failed")

	_, err = s.Client.Object.AbortMultipartUpload(context.Background(), name, v.UploadID)
	assert.Nil(s.T(), err, "AbortMultipartUpload Failed")
}

func (s *CosTestSuite) TestCreateCompleteMultipartUpload() {
	name := "test/test_complete_upload" + time.Now().Format(time.RFC3339)
	v, _, err := s.Client.Object.InitiateMultipartUpload(context.Background(), name, nil)
	uploadID := v.UploadID
	blockSize := 1024 * 1024 * 3

	opt := &cos.CompleteMultipartUploadOptions{}
	for i := 1; i < 3; i++ {
		b := make([]byte, blockSize)
		_, err := rand.Read(b)
		content := fmt.Sprintf("%X", b)
		f := strings.NewReader(content)

		resp, err := s.Client.Object.UploadPart(
			context.Background(), name, uploadID, i, f, nil,
		)
		assert.Nil(s.T(), err, "UploadPart Failed")
		etag := resp.Header.Get("Etag")
		opt.Parts = append(opt.Parts, cos.Object{
			PartNumber: i, ETag: etag},
		)
	}

	_, _, err = s.Client.Object.CompleteMultipartUpload(
		context.Background(), name, uploadID, opt,
	)

	assert.Nil(s.T(), err, "CompleteMultipartUpload Failed")
}

// End of api test

// All methods that begin with "Test" are run as tests within a
// suite.
// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCosTestSuite(t *testing.T) {
	suite.Run(t, new(CosTestSuite))
}

func (s *CosTestSuite) TearDownSuite() {
	// Clean the file in bucket
	// r, _, err := s.Client.Bucket.ListMultipartUploads(context.Background(), nil)
	// assert.Nil(s.T(), err, "ListMultipartUploads Failed")
	// for _, p := range r.Uploads {
	// 	// Abort
	// 	_, err = s.Client.Object.AbortMultipartUpload(context.Background(), p.Key, p.UploadID)
	// 	assert.Nil(s.T(), err, "AbortMultipartUpload Failed")
	// }

	// // Delete objects
	// opt := &cos.BucketGetOptions{
	// 	MaxKeys: 500,
	// }
	// v, _, err := s.Client.Bucket.Get(context.Background(), opt)
	// assert.Nil(s.T(), err, "GetBucket Failed")
	// for _, c := range v.Contents {
	// 	_, err := s.Client.Object.Delete(context.Background(), c.Key)
	// 	assert.Nil(s.T(), err, "DeleteObject Failed")
	// }

	// When clean up these infos, can not solve the concurrent test problem

	fmt.Println("tear down~")

}
