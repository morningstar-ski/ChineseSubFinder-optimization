package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/cron_helper"
	taskqueue "github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/emby"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	task_queue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
}

func newControllerTestTaskQueue(t *testing.T) *taskqueue.TaskQueue {
	t.Helper()

	cacheName := fmt.Sprintf("controller_jobs_%d", time.Now().UnixNano())
	cache_center.DelDb(cacheName)

	var queue *taskqueue.TaskQueue
	func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprint(r)
				if strings.Contains(msg, "go-sqlite3 requires cgo to work") {
					t.Skip("skip controller job tests: sqlite driver requires cgo in this environment")
				}
				panic(r)
			}
		}()
		queue = taskqueue.NewTaskQueue(cache_center.NewCacheCenter(cacheName, log_helper.GetLogger4Tester()))
	}()

	t.Cleanup(func() {
		queue.Close()
		cache_center.DelDb(cacheName)
	})

	return queue
}

func TestChangeJobStatusHandlerWaitingUsesManualRequeue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	queue := newControllerTestTaskQueue(t)
	job := *task_queue2.NewOneJob(common.Movie, "manual-requeue", taskqueue.DefaultTaskPriorityLevel)
	job.DownloadTimes = 1
	job.UpdateTime = emby.Time(time.Now())
	job.ErrorInfo = "all site download sub not found"

	bok, err := queue.Add(job)
	if err != nil {
		t.Fatal(err)
	}
	if bok == false {
		t.Fatal("expected job add to succeed")
	}

	cb := &ControllerBase{
		log:        logrus.New(),
		cronHelper: &cron_helper.CronHelper{DownloadQueue: queue},
	}

	reqBody, err := json.Marshal(backend2.ReqChangeJobStatus{
		Id:           job.Id,
		TaskPriority: "high",
		JobStatus:    task_queue2.Waiting,
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/jobs/change-job-status", bytes.NewReader(reqBody))
	ctx.Request.Header.Set("Content-Type", "application/json")

	cb.ChangeJobStatusHandler(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	found, waitingJob, err := queue.GetOneWaitingJob()
	if err != nil {
		t.Fatal(err)
	}
	if found == false {
		t.Fatal("expected manual requeue to make job immediately runnable")
	}
	if waitingJob.Id != job.Id {
		t.Fatalf("job id = %q, want %q", waitingJob.Id, job.Id)
	}
	if waitingJob.TaskPriority != taskqueue.HighTaskPriorityLevel {
		t.Fatalf("task priority = %d, want %d", waitingJob.TaskPriority, taskqueue.HighTaskPriorityLevel)
	}
	if waitingJob.DownloadTimes != 0 {
		t.Fatalf("download times = %d, want 0", waitingJob.DownloadTimes)
	}
	if waitingJob.RetryTimes != 0 {
		t.Fatalf("retry times = %d, want 0", waitingJob.RetryTimes)
	}
	if waitingJob.ErrorInfo != "" {
		t.Fatalf("error info = %q, want empty", waitingJob.ErrorInfo)
	}
}
