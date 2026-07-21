package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/scheduler"
)

type createScheduleRequest struct {
	Name       string   `json:"name" binding:"required"`
	SourceConn string   `json:"source_conn" binding:"required"`
	TargetConn string   `json:"target_conn" binding:"required"`
	Tables     []string `json:"tables"`
	OnConflict string   `json:"on_conflict"`
	BatchSize  int      `json:"batch_size"`
	CronExpr   string   `json:"cron_expr" binding:"required"`
	Enabled    bool     `json:"enabled"`
}

type updateScheduleRequest struct {
	Name       *string  `json:"name"`
	SourceConn *string  `json:"source_conn"`
	TargetConn *string  `json:"target_conn"`
	Tables     []string `json:"tables"`
	OnConflict *string  `json:"on_conflict"`
	BatchSize  *int     `json:"batch_size"`
	CronExpr   *string  `json:"cron_expr"`
	Enabled    *bool    `json:"enabled"`
}

func ListSchedules(sched *scheduler.Scheduler, store scheduler.SchedulerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobs, err := store.List()
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.Success(c, jobs)
	}
}

func GetSchedule(_ *scheduler.Scheduler, store scheduler.SchedulerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		job, err := store.Get(c.Param("id"))
		if err != nil {
			response.NotFound(c, "schedule not found")
			return
		}
		response.Success(c, job)
	}
}

func CreateSchedule(sched *scheduler.Scheduler, store scheduler.SchedulerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		onConflict := req.OnConflict
		if onConflict == "" {
			onConflict = "error"
		}
		batchSize := req.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}

		job := &scheduler.ScheduledJob{
			ID:         scheduler.GenID(),
			Name:       req.Name,
			SourceConn: req.SourceConn,
			TargetConn: req.TargetConn,
			Tables:     req.Tables,
			OnConflict: onConflict,
			BatchSize:  batchSize,
			CronExpr:   req.CronExpr,
			Enabled:    req.Enabled,
		}

		if err := sched.EnsureJob(job); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		response.Created(c, job)
	}
}

func UpdateSchedule(sched *scheduler.Scheduler, store scheduler.SchedulerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		existing, err := store.Get(c.Param("id"))
		if err != nil {
			response.NotFound(c, "schedule not found")
			return
		}

		var req updateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		if req.Name != nil {
			existing.Name = *req.Name
		}
		if req.SourceConn != nil {
			existing.SourceConn = *req.SourceConn
		}
		if req.TargetConn != nil {
			existing.TargetConn = *req.TargetConn
		}
		if req.Tables != nil {
			existing.Tables = req.Tables
		}
		if req.OnConflict != nil {
			existing.OnConflict = *req.OnConflict
		}
		if req.BatchSize != nil {
			existing.BatchSize = *req.BatchSize
		}
		if req.CronExpr != nil {
			existing.CronExpr = *req.CronExpr
		}
		if req.Enabled != nil {
			existing.Enabled = *req.Enabled
		}

		if err := sched.EnsureJob(existing); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		response.Success(c, existing)
	}
}

func DeleteSchedule(_ *scheduler.Scheduler, store scheduler.SchedulerStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.Delete(c.Param("id")); err != nil {
			response.NotFound(c, "schedule not found")
			return
		}
		c.Status(http.StatusNoContent)
	}
}
