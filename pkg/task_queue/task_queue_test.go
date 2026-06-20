package task_queue

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/emby"
	task_queue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

const taskQueueName = "testQueue"

func init() {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
}

func newTestTaskQueue(t *testing.T) *TaskQueue {
	t.Helper()

	cache_center.DelDb(taskQueueName)
	taskQueue := newTaskQueueOrSkip(t)
	t.Cleanup(func() {
		taskQueue.Close()
		cache_center.DelDb(taskQueueName)
	})

	return taskQueue
}

func newTaskQueueOrSkip(t *testing.T) (taskQueue *TaskQueue) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "go-sqlite3 requires cgo to work") {
				t.Skip("skip task_queue tests: sqlite driver requires cgo in this environment")
			}
			panic(r)
		}
	}()

	return NewTaskQueue(cache_center.NewCacheCenter(taskQueueName, log_helper.GetLogger4Tester()))
}

func TestTaskQueue_AddAndGetAndDel(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, pkg.RandStringBytesMaskImprSrcSB(10), i))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	bok, waitingJobs, err := taskQueue.GetJobsByStatus(task_queue2.Waiting)
	if err != nil {
		t.Fatal("TestTaskQueue.Get", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Get == false")
	}

	if len(waitingJobs) != taskPriorityCount+1 {
		t.Fatal("len(waitingJobs) != taskPriorityCount")
	}

	for i := 0; i <= taskPriorityCount; i++ {

		if waitingJobs[i].TaskPriority != i {
			t.Fatalf("TestTaskQueue.TaskPriority pop error, want = %d, got = %d", i, waitingJobs[i].TaskPriority)
		}
	}

	for _, waitingJob := range waitingJobs {
		bok, err = taskQueue.Del(waitingJob.Id)
		if err != nil {
			t.Fatal("TestTaskQueue.Del", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Del == false")
		}
	}

	if taskQueue.Size() != 0 {
		t.Fatal("taskQueue.Size() != 0")
	}
}

func TestTaskQueue_PreservesChineseVideoPath(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	videoPath := "/media/movies/外语电影/记忆碎片 (2000)/记忆碎片 (2000) - 1080p.mkv"

	bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, videoPath, DefaultTaskPriorityLevel))
	if err != nil {
		t.Fatal("TestTaskQueue.Add", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Add == false")
	}

	bok, jobs, err := taskQueue.GetAllJobs()
	if err != nil {
		t.Fatal("TestTaskQueue.GetAllJobs", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetAllJobs == false")
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if jobs[0].VideoFPath != videoPath {
		t.Fatalf("VideoFPath = %q, want %q", jobs[0].VideoFPath, videoPath)
	}
	if jobs[0].VideoName != "记忆碎片 (2000) - 1080p.mkv" {
		t.Fatalf("VideoName = %q", jobs[0].VideoName)
	}

	taskQueue.Close()

	reloaded := newTaskQueueOrSkip(t)
	t.Cleanup(func() {
		reloaded.Close()
		cache_center.DelDb(taskQueueName)
	})

	bok, jobs, err = reloaded.GetAllJobs()
	if err != nil {
		t.Fatal("Reloaded.GetAllJobs", err)
	}
	if bok == false {
		t.Fatal("Reloaded.GetAllJobs == false")
	}
	if len(jobs) != 1 {
		t.Fatalf("len(reloaded jobs) = %d, want 1", len(jobs))
	}
	if jobs[0].VideoFPath != videoPath {
		t.Fatalf("reloaded VideoFPath = %q, want %q", jobs[0].VideoFPath, videoPath)
	}
	if jobs[0].VideoName != "记忆碎片 (2000) - 1080p.mkv" {
		t.Fatalf("reloaded VideoName = %q", jobs[0].VideoName)
	}
}

func TestTaskQueue_AddAndClear(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, pkg.RandStringBytesMaskImprSrcSB(10), i))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	err := taskQueue.Clear()
	if err != nil {
		t.Fatal("TestTaskQueue.Clear", err)
	}

	if taskQueue.Size() != 0 {
		t.Fatal("taskQueue.Size() != 0")
	}
}

func TestTaskQueue_Update(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, pkg.RandStringBytesMaskImprSrcSB(10), i))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	bok, waitingJobs, err := taskQueue.GetJobsByStatus(task_queue2.Waiting)
	if err != nil {
		t.Fatal("TestTaskQueue.Get", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Get == false")
	}

	if len(waitingJobs) != taskPriorityCount+1 {
		t.Fatal("len(waitingJobs) != taskPriorityCount")
	}

	for i := 0; i <= taskPriorityCount; i++ {

		if waitingJobs[i].TaskPriority != i {
			t.Fatalf("TestTaskQueue.TaskPriority pop error, want = %d, got = %d", i, waitingJobs[i].TaskPriority)
		}
	}

	for _, waitingJob := range waitingJobs {

		waitingJob.JobStatus = task_queue2.Committed

		bok, err = taskQueue.Update(waitingJob)
		if err != nil {
			t.Fatal("TestTaskQueue.Update", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Update == false")
		}
	}

	bok, committedJobs, err := taskQueue.GetJobsByStatus(task_queue2.Committed)
	if err != nil {
		t.Fatal("TestTaskQueue.Get", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Get == false")
	}

	if len(committedJobs) != taskPriorityCount+1 {
		t.Fatal("len(committedJobs) != taskPriorityCount")
	}
}

func TestTaskQueue_UpdateAdGetOneWaiting(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, fmt.Sprintf("%d", i), i))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	bok, waitingJob, err := taskQueue.GetOneWaitingJob()
	if err != nil {
		t.Fatal("TestTaskQueue.GetOneWaitingJob", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetOneWaitingJob == false")
	}

	if waitingJob.TaskPriority != 0 {
		t.Fatal("waitingJob.TaskPriority != 0")
	}

	waitingJob.JobStatus = task_queue2.Committed
	bok, err = taskQueue.Update(waitingJob)
	if err != nil {
		t.Fatal("TestTaskQueue.Update", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Update == false")
	}

	bok, waitingJob, err = taskQueue.GetOneWaitingJob()
	if err != nil {
		t.Fatal("TestTaskQueue.GetOneWaitingJob", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetOneWaitingJob == false")
	}

	if waitingJob.TaskPriority != 1 {
		t.Fatal("waitingJob.TaskPriority != 0")
	}
}

func TestTaskQueue_UpdatePriority(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, fmt.Sprintf("%d", i), i))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	bok, waitingJob, err := taskQueue.GetOneWaitingJob()
	if err != nil {
		t.Fatal("TestTaskQueue.GetOneWaitingJob", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetOneWaitingJob == false")
	}

	if waitingJob.TaskPriority != 0 {
		t.Fatal("waitingJob.TaskPriority != 0")
	}

	waitingJob.TaskPriority = 1
	bok, err = taskQueue.Update(waitingJob)
	if err != nil {
		t.Fatal("TestTaskQueue.Update", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Update == false")
	}

	bok, waitingJobs, err := taskQueue.GetJobsByPriorityAndStatus(0, task_queue2.Waiting)
	if err != nil {
		t.Fatal("TestTaskQueue.GetJobsByPriorityAndStatus", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetJobsByPriorityAndStatus == false")
	}

	if len(waitingJobs) != 0 {
		t.Fatal("len(waitingJobs) != 0")
	}

	bok, waitingJobs, err = taskQueue.GetJobsByPriorityAndStatus(1, task_queue2.Waiting)
	if err != nil {
		t.Fatal("TestTaskQueue.GetJobsByPriorityAndStatus", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.GetJobsByPriorityAndStatus == false")
	}

	if len(waitingJobs) != 2 {
		t.Fatal("len(waitingJobs) != 2")
	}
}

func TestTaskQueue_RequeueForManualTriggerMakesJobImmediatelyRunnable(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	nowJob := *task_queue2.NewOneJob(common.Movie, "manual-requeue", DefaultTaskPriorityLevel)
	nowJob.DownloadTimes = 1
	nowJob.UpdateTime = emby.Time(time.Now())
	nowJob.ErrorInfo = "all site download sub not found"

	bok, err := taskQueue.Add(nowJob)
	if err != nil {
		t.Fatal("TaskQueue.Add", err)
	}
	if bok == false {
		t.Fatal("TaskQueue.Add == false")
	}

	bok, waitingJob, err := taskQueue.GetOneWaitingJob()
	if err != nil {
		t.Fatal("TaskQueue.GetOneWaitingJob", err)
	}
	if bok == true {
		t.Fatal("expected cooldown to block immediate pickup before manual requeue")
	}
	if waitingJob.Id != "" {
		t.Fatal("waitingJob should be empty before manual requeue")
	}

	nowJob.TaskPriority = HighTaskPriorityLevel
	bok, err = taskQueue.RequeueForManualTrigger(nowJob)
	if err != nil {
		t.Fatal("TaskQueue.RequeueForManualTrigger", err)
	}
	if bok == false {
		t.Fatal("TaskQueue.RequeueForManualTrigger == false")
	}

	bok, waitingJob, err = taskQueue.GetOneWaitingJob()
	if err != nil {
		t.Fatal("TaskQueue.GetOneWaitingJob", err)
	}
	if bok == false {
		t.Fatal("expected manual requeue to make job immediately runnable")
	}
	if waitingJob.TaskPriority != HighTaskPriorityLevel {
		t.Fatalf("TaskPriority = %d, want %d", waitingJob.TaskPriority, HighTaskPriorityLevel)
	}
	if waitingJob.DownloadTimes != 0 {
		t.Fatalf("DownloadTimes = %d, want 0", waitingJob.DownloadTimes)
	}
	if waitingJob.RetryTimes != 0 {
		t.Fatalf("RetryTimes = %d, want 0", waitingJob.RetryTimes)
	}
	if waitingJob.JobStatus != task_queue2.Waiting {
		t.Fatalf("JobStatus = %v, want Waiting", waitingJob.JobStatus)
	}
	if waitingJob.ErrorInfo != "" {
		t.Fatalf("ErrorInfo = %q, want empty", waitingJob.ErrorInfo)
	}
}

func TestTaskQueue_AutoDetectUpdateJobStatusMarksNoSubFoundAsFailed(t *testing.T) {

	taskQueue := newTestTaskQueue(t)
	nowJob := *task_queue2.NewOneJob(common.Movie, "no-sub-found", DefaultTaskPriorityLevel)

	bok, err := taskQueue.Add(nowJob)
	if err != nil {
		t.Fatal("TaskQueue.Add", err)
	}
	if bok == false {
		t.Fatal("TaskQueue.Add == false")
	}

	taskQueue.AutoDetectUpdateJobStatus(nowJob, ErrNoSubFound)

	bok, gotJob := taskQueue.GetOneJobByID(nowJob.Id)
	if bok == false {
		t.Fatal("TaskQueue.GetOneJobByID == false")
	}
	if gotJob.JobStatus != task_queue2.Failed {
		t.Fatalf("JobStatus = %v, want Failed", gotJob.JobStatus)
	}
	if gotJob.ErrorInfo != ErrNoSubFound.Error() {
		t.Fatalf("ErrorInfo = %q, want %q", gotJob.ErrorInfo, ErrNoSubFound.Error())
	}
	if gotJob.DownloadTimes != 1 {
		t.Fatalf("DownloadTimes = %d, want 1", gotJob.DownloadTimes)
	}
	if gotJob.TaskPriority != DefaultTaskPriorityLevel {
		t.Fatalf("TaskPriority = %d, want %d", gotJob.TaskPriority, DefaultTaskPriorityLevel)
	}
}

func TestTaskQueue_AddAndGetOneJob(t *testing.T) {

	taskQueue := newTestTaskQueue(t)

	for i := taskPriorityCount; i >= 0; i-- {
		bok, err := taskQueue.Add(*task_queue2.NewOneJob(common.Movie, fmt.Sprintf("%d", i), DefaultTaskPriorityLevel))
		if err != nil {
			t.Fatal("TestTaskQueue.Add", err)
		}
		if bok == false {
			t.Fatal("TestTaskQueue.Add == false")
		}
	}

	bok, oneJob, err := taskQueue.GetOneJob()
	if err != nil {
		t.Fatal("TestTaskQueue.Add", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Add == false")
	}

	println("VideoFPath", oneJob.VideoFPath)
	println("TaskPriority", oneJob.TaskPriority)

	taskQueue.AutoDetectUpdateJobStatus(oneJob, nil)

	bok, oneJob, err = taskQueue.GetOneJob()
	if err != nil {
		t.Fatal("TestTaskQueue.Add", err)
	}
	if bok == false {
		t.Fatal("TestTaskQueue.Add == false")
	}

	println("VideoFPath", oneJob.VideoFPath)
	println("TaskPriority", oneJob.TaskPriority)

	found, waitingJobs, err := taskQueue.GetJobsByStatus(task_queue2.Waiting)
	if err != nil {
		return
	}
	println(found)
	for i, job := range waitingJobs {
		println("QueueDownloader Waiting:", i, job.VideoName)
	}

	found, waitingJobs, err = taskQueue.GetJobsByStatus(task_queue2.Done)
	if err != nil {
		return
	}
	println(found)
	for i, job := range waitingJobs {
		println("QueueDownloader Done:", i, job.VideoName)
	}

	found, waitingJobs, err = taskQueue.GetJobsByStatus(task_queue2.Failed)
	if err != nil {
		return
	}
	println(found)
	for i, job := range waitingJobs {
		println("QueueDownloader Failed:", i, job.VideoName)
	}

}
