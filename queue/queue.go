package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/techagentng/iweapp/models"
)

const (
	// Redis keys
	JobQueueKey         = "iwe:jobs:queue"          // List for job queue (FIFO)
	JobPriorityQueueKey = "iwe:jobs:priority_queue" // Sorted set for priority queue
	JobKeyPrefix        = "iwe:job:"                // Hash for job details
	UserJobsKeyPrefix   = "iwe:user:jobs:"          // Set of job IDs per user
	JobTTL              = 24 * time.Hour            // Job data retention
)

// QueueManager handles job queue operations using Redis
type QueueManager struct {
	redis *redis.Client
	ctx   context.Context
}

// NewQueueManager creates a new queue manager instance
func NewQueueManager(redisClient *redis.Client) *QueueManager {
	return &QueueManager{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// EnqueueJob adds a new job to the processing queue
func (qm *QueueManager) EnqueueJob(job *models.ProcessingJob) error {
	// Serialize job to JSON
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Store job details in Redis hash
	jobKey := fmt.Sprintf("%s%s", JobKeyPrefix, job.ID.String())
	if err := qm.redis.Set(qm.ctx, jobKey, jobJSON, JobTTL).Err(); err != nil {
		return fmt.Errorf("failed to store job: %w", err)
	}

	// Add job ID to user's job set
	userJobsKey := fmt.Sprintf("%s%s", UserJobsKeyPrefix, job.UserID.String())
	if err := qm.redis.SAdd(qm.ctx, userJobsKey, job.ID.String()).Err(); err != nil {
		log.Printf("Warning: failed to add job to user set: %v", err)
	}
	qm.redis.Expire(qm.ctx, userJobsKey, JobTTL)

	// Enqueue based on priority
	if job.Priority > 0 {
		// Use priority queue (sorted set) - higher priority = lower score for ZPOPMIN
		score := float64(-job.Priority) // Negative so higher priority comes first
		if err := qm.redis.ZAdd(qm.ctx, JobPriorityQueueKey, redis.Z{
			Score:  score,
			Member: job.ID.String(),
		}).Err(); err != nil {
			return fmt.Errorf("failed to enqueue priority job: %w", err)
		}
		log.Printf("📋 Priority job enqueued: %s (priority: %d, user: %s)", job.ID, job.Priority, job.UserID)
	} else {
		// Use regular FIFO queue
		if err := qm.redis.RPush(qm.ctx, JobQueueKey, job.ID.String()).Err(); err != nil {
			return fmt.Errorf("failed to enqueue job: %w", err)
		}
		log.Printf("📋 Job enqueued: %s (user: %s, file: %s)", job.ID, job.UserID, job.FileName)
	}

	return nil
}

// DequeueJob retrieves the next job from the queue (blocking operation)
// Checks priority queue first, then regular queue
func (qm *QueueManager) DequeueJob(timeout time.Duration) (*models.ProcessingJob, error) {
	// First, check priority queue (non-blocking)
	priorityResult, err := qm.redis.ZPopMin(qm.ctx, JobPriorityQueueKey, 1).Result()
	if err == nil && len(priorityResult) > 0 {
		jobID := priorityResult[0].Member.(string)
		job, err := qm.GetJob(uuid.MustParse(jobID))
		if err == nil {
			// Check if job is ready to run (for scheduled jobs)
			if job.IsReadyToRun() {
				return job, nil
			}
			// Not ready yet, re-enqueue
			qm.redis.ZAdd(qm.ctx, JobPriorityQueueKey, redis.Z{
				Score:  priorityResult[0].Score,
				Member: jobID,
			})
		}
	}

	// Then check regular queue (blocking with timeout)
	result, err := qm.redis.BLPop(qm.ctx, timeout, JobQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue result")
	}

	jobID := result[1]

	// Retrieve job details
	job, err := qm.GetJob(uuid.MustParse(jobID))
	if err != nil {
		return nil, err
	}

	// Check if scheduled job is ready
	if !job.IsReadyToRun() {
		// Re-enqueue for later
		qm.redis.RPush(qm.ctx, JobQueueKey, jobID)
		return nil, nil
	}

	return job, nil
}

// GetJob retrieves a job by its ID
func (qm *QueueManager) GetJob(jobID uuid.UUID) (*models.ProcessingJob, error) {
	jobKey := fmt.Sprintf("%s%s", JobKeyPrefix, jobID.String())
	
	jobJSON, err := qm.redis.Get(qm.ctx, jobKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job models.ProcessingJob
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates job details in Redis
func (qm *QueueManager) UpdateJob(job *models.ProcessingJob) error {
	jobKey := fmt.Sprintf("%s%s", JobKeyPrefix, job.ID.String())
	
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := qm.redis.Set(qm.ctx, jobKey, jobJSON, JobTTL).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// UpdateJobStatus updates the status and progress of a job
func (qm *QueueManager) UpdateJobStatus(jobID uuid.UUID, status models.JobStatus, progress int) error {
	job, err := qm.GetJob(jobID)
	if err != nil {
		return err
	}

	// Update status and progress
	job.Status = status
	job.Progress = progress

	// Update timestamps
	now := time.Now()
	if status == models.JobStatusProcessing && job.StartedAt == nil {
		job.StartedAt = &now
	}
	if status == models.JobStatusCompleted || status == models.JobStatusFailed {
		job.CompletedAt = &now
	}

	return qm.UpdateJob(job)
}

// UpdateJobProgress updates just the progress percentage
func (qm *QueueManager) UpdateJobProgress(jobID uuid.UUID, progress int, message string) error {
	job, err := qm.GetJob(jobID)
	if err != nil {
		return err
	}

	job.Progress = progress
	
	if err := qm.UpdateJob(job); err != nil {
		return err
	}

	// Publish progress update
	return qm.PublishJobUpdate(job.UserID, job)
}

// SetJobError sets error message for a failed job
func (qm *QueueManager) SetJobError(jobID uuid.UUID, errorMsg string) error {
	job, err := qm.GetJob(jobID)
	if err != nil {
		return err
	}

	job.Status = models.JobStatusFailed
	job.ErrorMsg = errorMsg
	job.RetryCount++
	
	now := time.Now()
	job.CompletedAt = &now

	if err := qm.UpdateJob(job); err != nil {
		return err
	}

	// Publish failure notification
	return qm.PublishJobUpdate(job.UserID, job)
}

// PublishJobUpdate publishes job update to user's Redis pub/sub channel
func (qm *QueueManager) PublishJobUpdate(userID uuid.UUID, job *models.ProcessingJob) error {
	channel := fmt.Sprintf("user:%s:jobs", userID.String())
	
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job for publish: %w", err)
	}

	if err := qm.redis.Publish(qm.ctx, channel, jobJSON).Err(); err != nil {
		return fmt.Errorf("failed to publish job update: %w", err)
	}

	return nil
}

// GetUserJobs retrieves all job IDs for a user
func (qm *QueueManager) GetUserJobs(userID uuid.UUID) ([]uuid.UUID, error) {
	userJobsKey := fmt.Sprintf("%s%s", UserJobsKeyPrefix, userID.String())
	
	jobIDs, err := qm.redis.SMembers(qm.ctx, userJobsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user jobs: %w", err)
	}

	var jobs []uuid.UUID
	for _, idStr := range jobIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			jobs = append(jobs, id)
		}
	}

	return jobs, nil
}

// GetQueueLength returns the number of jobs waiting in the queue
func (qm *QueueManager) GetQueueLength() (int64, error) {
	length, err := qm.redis.LLen(qm.ctx, JobQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}

// DeleteJob removes a job from Redis
func (qm *QueueManager) DeleteJob(jobID uuid.UUID) error {
	jobKey := fmt.Sprintf("%s%s", JobKeyPrefix, jobID.String())
	
	if err := qm.redis.Del(qm.ctx, jobKey).Err(); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

// SubscribeToUserJobs subscribes to job updates for a specific user
func (qm *QueueManager) SubscribeToUserJobs(userID uuid.UUID) *redis.PubSub {
	channel := fmt.Sprintf("user:%s:jobs", userID.String())
	return qm.redis.Subscribe(qm.ctx, channel)
}
