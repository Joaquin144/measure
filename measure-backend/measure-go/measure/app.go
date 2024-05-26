package measure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"measure-backend/measure-go/event"
	"measure-backend/measure-go/filter"
	"measure-backend/measure-go/group"
	"measure-backend/measure-go/journey"
	"measure-backend/measure-go/metrics"
	"measure-backend/measure-go/replay"
	"measure-backend/measure-go/server"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/leporo/sqlf"
)

type App struct {
	ID           *uuid.UUID `json:"id"`
	TeamId       uuid.UUID  `json:"team_id"`
	AppName      string     `json:"name" binding:"required"`
	UniqueId     string     `json:"unique_identifier"`
	Platform     string     `json:"platform"`
	APIKey       *APIKey    `json:"api_key"`
	FirstVersion string     `json:"first_version"`
	Onboarded    bool       `json:"onboarded"`
	OnboardedAt  time.Time  `json:"onboarded_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (a App) MarshalJSON() ([]byte, error) {
	type Alias App
	return json.Marshal(&struct {
		*Alias
		Platform    *string    `json:"platform"`
		OnboardedAt *time.Time `json:"onboarded_at"`
		UniqueId    *string    `json:"unique_identifier"`
	}{
		Platform: func() *string {
			if a.Platform == "" {
				return nil
			}
			return &a.Platform
		}(),
		UniqueId: func() *string {
			if a.UniqueId == "" {
				return nil
			}
			return &a.UniqueId
		}(),
		OnboardedAt: func() *time.Time {
			if a.OnboardedAt.IsZero() {
				return nil
			}
			return &a.OnboardedAt
		}(),
		Alias: (*Alias)(&a),
	})
}

// GetExceptionGroup queries a single exception group from the exception
// group id and returns a pointer to ExceptionGroup.
func (a App) GetExceptionGroup(ctx context.Context, id uuid.UUID) (*group.ExceptionGroup, error) {
	stmt := sqlf.PostgreSQL.
		Select("id, app_id, name, fingerprint, array_length(event_ids, 1) as count, event_ids, created_at, updated_at").
		From("unhandled_exception_groups").
		Where("id = ?", nil)
	defer stmt.Close()

	rows, err := server.Server.PgPool.Query(ctx, stmt.String(), id)
	if err != nil {
		return nil, err
	}
	group, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[group.ExceptionGroup])
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// GetANRGroup queries a single anr group from the anr
// group id and returns a pointer to ANRGroup.
func (a App) GetANRGroup(ctx context.Context, id uuid.UUID) (*group.ANRGroup, error) {
	stmt := sqlf.PostgreSQL.
		Select("id, app_id, name, fingerprint, array_length(event_ids, 1) as count, event_ids, created_at, updated_at").
		From("anr_groups").
		Where("id = ?", nil)
	defer stmt.Close()

	rows, err := server.Server.PgPool.Query(ctx, stmt.String(), id)
	if err != nil {
		return nil, err
	}
	group, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[group.ANRGroup])
	if err != nil {
		return nil, err
	}

	return &group, nil
}

// GetExceptionGroups returns slice of ExceptionGroup after applying matching
// AppFilter values
func (a App) GetExceptionGroups(ctx context.Context, af *filter.AppFilter) ([]group.ExceptionGroup, error) {
	stmt := sqlf.PostgreSQL.
		Select("id, app_id, name, fingerprint, array_length(event_ids, 1) as count, event_ids, created_at, updated_at").
		From("public.unhandled_exception_groups").
		OrderBy("count desc").
		Where("app_id = ?", nil)

	defer stmt.Close()

	args := []any{a.ID}

	if af != nil {
		if af.HasTimeRange() {
			stmt.Where("created_at >= ? and created_at <= ?", nil, nil)
			args = append(args, af.From, af.To)
		}
	}

	rows, err := server.Server.PgPool.Query(ctx, stmt.String(), args...)
	if err != nil {
		return nil, err
	}
	groups, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[group.ExceptionGroup])
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// GetANRGroups returns slice of ANRGroup after applying matching
// AppFilter values
func (a App) GetANRGroups(af *filter.AppFilter) ([]group.ANRGroup, error) {
	stmt := sqlf.PostgreSQL.
		Select("id, app_id, name, fingerprint, array_length(event_ids, 1) as count, event_ids, created_at, updated_at").
		From("public.anr_groups").
		OrderBy("count desc").
		Where("app_id = ?", nil)
	defer stmt.Close()

	args := []any{a.ID}

	if af != nil {
		if af.HasTimeRange() {
			stmt.Where("created_at >= ? and created_at <= ?", nil, nil)
			args = append(args, af.From, af.To)
		}
	}

	rows, err := server.Server.PgPool.Query(context.Background(), stmt.String(), args...)
	if err != nil {
		return nil, err
	}
	groups, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[group.ANRGroup])
	if err != nil {
		return nil, err
	}

	return groups, nil
}

func (a App) GetSizeMetrics(ctx context.Context, af *filter.AppFilter) (size *metrics.SizeMetric, err error) {
	size = &metrics.SizeMetric{}
	stmt := sqlf.Select("count(id) as count").
		From("default.events").
		Where("app_id = ?", nil).
		Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil).
		Where("timestamp >= ? and timestamp <= ?", nil, nil)

	defer stmt.Close()

	args := []any{a.ID, af.Versions[0], af.VersionCodes[0], af.From, af.To}
	var count uint64

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&count); err != nil {
		return nil, err
	}

	// no events for selected conditions found
	if count < 1 {
		size.SetNaNs()
		return
	}

	sizeStmt := sqlf.PostgreSQL.
		With("avg_size",
			sqlf.PostgreSQL.From("public.build_sizes").
				Select("round(avg(build_size), 2) as average_size").
				Where("app_id = ?", nil)).
		Select("t1.average_size as average_app_size").
		Select("t2.build_size as selected_app_size").
		Select("(t2.build_size - t1.average_size) as delta").
		From("avg_size as t1, public.build_sizes as t2").
		Where("app_id = ?", nil).
		Where("version_name = ?", nil).
		Where("version_code = ?", nil)

	defer sizeStmt.Close()

	args = []any{a.ID, a.ID, af.Versions[0], af.VersionCodes[0]}

	ctx = context.Background()
	if err := server.Server.PgPool.QueryRow(ctx, sizeStmt.String(), args...).Scan(&size.AverageAppSize, &size.SelectedAppSize, &size.Delta); err != nil {
		return nil, err
	}

	return
}

func (a App) GetCrashFreeMetrics(ctx context.Context, af *filter.AppFilter) (crashFree *metrics.CrashFreeSession, err error) {
	crashFree = &metrics.CrashFreeSession{}
	stmt := sqlf.
		With("all_sessions",
			sqlf.From("default.events").
				Select("session_id, attribute.app_version, attribute.app_build, type, exception.handled").
				Where(`app_id = ? and timestamp >= ? and timestamp <= ?`, nil, nil, nil)).
		With("t1",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as total_sessions_selected").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t2",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_exception_selected").
				Where("`type` = 'exception' and `exception.handled` = false").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t3",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_exception").
				Where("`type` != 'exception'")).
		With("t4",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_exception_selected").
				Where("`type` != 'exception'").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		Select("round((1 - (t2.count_exception_selected / t1.total_sessions_selected)) * 100, 2) as crash_free_sessions").
		Select("round(((t4.count_not_exception_selected - t3.count_not_exception) / t3.count_not_exception) * 100, 2) as delta").
		From("t1, t2, t3, t4")

	defer stmt.Close()

	version := af.Versions[0]
	code := af.VersionCodes[0]

	args := []any{a.ID, af.From, af.To, version, code, version, code, version, code}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&crashFree.CrashFreeSessions, &crashFree.Delta); err != nil {
		return nil, err
	}

	crashFree.SetNaNs()

	return
}

func (a App) GetANRFreeMetrics(ctx context.Context, af *filter.AppFilter) (anrFree *metrics.ANRFreeSession, err error) {
	anrFree = &metrics.ANRFreeSession{}
	stmt := sqlf.
		With("all_sessions",
			sqlf.From("default.events").
				Select("session_id, attribute.app_version, attribute.app_build, type").
				Where(`app_id = ? and timestamp >= ? and timestamp <= ?`, nil, nil, nil)).
		With("t1",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as total_sessions_selected").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t2",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_anr_selected").
				Where("`type` = 'anr'").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t3",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_anr").
				Where("`type` != 'anr'")).
		With("t4",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_anr_selected").
				Where("`type` != 'anr'").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		Select("round((1 - (t2.count_anr_selected / t1.total_sessions_selected)) * 100, 2) as anr_free_sessions").
		Select("round(((t4.count_not_anr_selected - t3.count_not_anr) / t3.count_not_anr) * 100, 2) as delta").
		From("t1, t2, t3, t4")

	defer stmt.Close()

	version := af.Versions[0]
	code := af.VersionCodes[0]

	args := []any{a.ID, af.From, af.To, version, code, version, code, version, code}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&anrFree.ANRFreeSessions, &anrFree.Delta); err != nil {
		return nil, err
	}

	anrFree.SetNaNs()

	return
}

func (a App) GetPerceivedCrashFreeMetrics(ctx context.Context, af *filter.AppFilter) (crashFree *metrics.PerceivedCrashFreeSession, err error) {
	crashFree = &metrics.PerceivedCrashFreeSession{}
	stmt := sqlf.
		With("all_sessions",
			sqlf.From("default.events").
				Select("session_id, attribute.app_version, attribute.app_build, type, exception.handled, exception.foreground").
				Where(`app_id = ? and timestamp >= ? and timestamp <= ?`, nil, nil, nil)).
		With("t1",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as total_sessions_selected").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t2",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_exception_selected").
				Where("`type` = 'exception' and `exception.handled` = false and `exception.foreground` = true").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t3",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_exception").
				Where("`type` != 'exception'")).
		With("t4",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_exception_selected").
				Where("`type` != 'exception'").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		Select("round((1 - (t2.count_exception_selected / t1.total_sessions_selected)) * 100, 2) as crash_free_sessions").
		Select("round(((t4.count_not_exception_selected - t3.count_not_exception) / t3.count_not_exception) * 100, 2) as delta").
		From("t1, t2, t3, t4")

	defer stmt.Close()

	version := af.Versions[0]
	code := af.VersionCodes[0]

	args := []any{a.ID, af.From, af.To, version, code, version, code, version, code}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&crashFree.CrashFreeSessions, &crashFree.Delta); err != nil {
		return nil, err
	}

	crashFree.SetNaNs()

	return
}

func (a App) GetPerceivedANRFreeMetrics(ctx context.Context, af *filter.AppFilter) (anrFree *metrics.PerceivedANRFreeSession, err error) {
	anrFree = &metrics.PerceivedANRFreeSession{}
	stmt := sqlf.
		With("all_sessions",
			sqlf.From("default.events").
				Select("session_id, attribute.app_version, attribute.app_build, type, anr.foreground").
				Where(`app_id = ? and timestamp >= ? and timestamp <= ?`, nil, nil, nil)).
		With("t1",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as total_sessions_selected").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t2",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_anr_selected").
				Where("`type` = 'anr' and `anr.foreground` = true").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		With("t3",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_anr").
				Where("`type` != 'anr'")).
		With("t4",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as count_not_anr_selected").
				Where("`type` != 'anr'").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		Select("round((1 - (t2.count_anr_selected / t1.total_sessions_selected)) * 100, 2) as anr_free_sessions").
		Select("round(((t4.count_not_anr_selected - t3.count_not_anr) / t3.count_not_anr) * 100, 2) as delta").
		From("t1, t2, t3, t4")

	defer stmt.Close()

	version := af.Versions[0]
	code := af.VersionCodes[0]

	args := []any{a.ID, af.From, af.To, version, code, version, code, version, code}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&anrFree.ANRFreeSessions, &anrFree.Delta); err != nil {
		return nil, err
	}

	anrFree.SetNaNs()

	return
}

func (a App) GetAdoptionMetrics(ctx context.Context, af *filter.AppFilter) (adoption *metrics.SessionAdoption, err error) {
	adoption = &metrics.SessionAdoption{}
	stmt := sqlf.From("default.events").
		With("all_sessions",
			sqlf.From("default.events").
				Select("session_id, attribute.app_version, attribute.app_build").
				Where(`app_id = ? and timestamp >= ? and timestamp <= ?`, nil, nil, nil)).
		With("all_versions",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as all_app_versions")).
		With("selected_version",
			sqlf.From("all_sessions").
				Select("count(distinct session_id) as selected_app_version").
				Where("`attribute.app_version` = ? and `attribute.app_build` = ?", nil, nil)).
		Select("t1.all_app_versions as all_app_versions", nil).
		Select("t2.selected_app_version as selected_app_version", nil).
		Select("round((t2.selected_app_version/t1.all_app_versions) * 100, 2) as adoption").
		From("all_versions as t1, selected_version as t2")

	defer stmt.Close()

	args := []any{a.ID, af.From, af.To, af.Versions[0], af.VersionCodes[0]}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&adoption.AllVersions, &adoption.SelectedVersion, &adoption.Adoption); err != nil {
		return nil, err
	}

	adoption.SetNaNs()

	return
}

func (a App) GetLaunchMetrics(ctx context.Context, af *filter.AppFilter) (launch *metrics.LaunchMetric, err error) {
	launch = &metrics.LaunchMetric{}
	stmt := sqlf.
		With("timings",
			sqlf.From("default.events").
				Select("type, cold_launch.duration, warm_launch.duration, hot_launch.duration, attribute.app_version, attribute.app_build").
				Where("app_id = ?", nil).
				Where("timestamp >= ? and timestamp <= ?", nil, nil).
				Where("(type = 'cold_launch' or type = 'warm_launch' or type = 'hot_launch')")).
		With("cold",
			sqlf.From("timings").
				Select("round(quantile(0.95)(cold_launch.duration), 2) as cold_launch").
				Where("type = 'cold_launch' and cold_launch.duration > 0")).
		With("warm",
			sqlf.From("timings").
				Select("round(quantile(0.95)(warm_launch.duration), 2) as warm_launch").
				Where("type = 'warm_launch' and warm_launch.duration > 0")).
		With("hot",
			sqlf.From("timings").
				Select("round(quantile(0.95)(hot_launch.duration), 2) as hot_launch").
				Where("type = 'hot_launch' and hot_launch.duration > 0")).
		With("cold_selected",
			sqlf.From("timings").
				Select("round(quantile(0.95)(cold_launch.duration), 2) as cold_launch").
				Where("type = 'cold_launch'").
				Where("cold_launch.duration > 0").
				Where("attribute.app_version = ? and attribute.app_build = ?", nil, nil)).
		With("warm_selected",
			sqlf.From("timings").
				Select("round(quantile(0.95)(warm_launch.duration), 2) as warm_launch").
				Where("type = 'warm_launch'").
				Where("warm_launch.duration > 0").
				Where("attribute.app_version = ? and attribute.app_build = ?", nil, nil)).
		With("hot_selected",
			sqlf.From("timings").
				Select("round(quantile(0.95)(hot_launch.duration), 2) as hot_launch").
				Where("type = 'hot_launch'").
				Where("hot_launch.duration > 0").
				Where("attribute.app_version = ? and attribute.app_build = ?", nil, nil)).
		Select("cold_selected.cold_launch as cold_launch_p95").
		Select("warm_selected.warm_launch as warm_launch_p95").
		Select("hot_selected.hot_launch as hot_launch_p95").
		Select("round(cold_selected.cold_launch - cold.cold_launch, 2) as cold_delta").
		Select("round(warm_selected.warm_launch - warm.warm_launch, 2) as warm_delta").
		Select("round(hot_selected.hot_launch - hot.hot_launch, 2) as hot_delta").
		From("cold, warm, hot, cold_selected, warm_selected, hot_selected")

	defer stmt.Close()

	version := af.Versions[0]
	code := af.VersionCodes[0]
	args := []any{a.ID, af.From, af.To, version, code, version, code, version, code}

	if err := server.Server.ChPool.QueryRow(ctx, stmt.String(), args...).Scan(&launch.ColdLaunchP95, &launch.WarmLaunchP95, &launch.HotLaunchP95, &launch.ColdDelta, &launch.WarmDelta, &launch.ColdDelta); err != nil {
		return nil, err
	}

	launch.SetNaNs()

	return
}

func (a App) GetSessionEvents(ctx context.Context, sessionId uuid.UUID) (*Session, error) {
	cols := []string{
		`id`,
		`toString(type)`,
		`session_id`,
		`app_id`,
		`inet.ipv4`,
		`inet.ipv6`,
		`inet.country_code`,
		`timestamp`,
		`attribute.installation_id`,
		`toString(attribute.app_version)`,
		`toString(attribute.app_build)`,
		`toString(attribute.app_unique_id)`,
		`toString(attribute.platform)`,
		`toString(attribute.measure_sdk_version)`,
		`toString(attribute.thread_name)`,
		`toString(attribute.user_id)`,
		`toString(attribute.device_name)`,
		`toString(attribute.device_model)`,
		`toString(attribute.device_manufacturer)`,
		`toString(attribute.device_type)`,
		`attribute.device_is_foldable`,
		`attribute.device_is_physical`,
		`attribute.device_density_dpi`,
		`attribute.device_width_px`,
		`attribute.device_height_px`,
		`attribute.device_density`,
		`toString(attribute.device_locale)`,
		`toString(attribute.os_name)`,
		`toString(attribute.os_version)`,
		`toString(attribute.network_type)`,
		`toString(attribute.network_generation)`,
		`toString(attribute.network_provider)`,
		`anr.fingerprint`,
		`anr.foreground`,
		`anr.exceptions`,
		`anr.threads`,
		`exception.handled`,
		`exception.fingerprint`,
		`exception.foreground`,
		`exception.exceptions`,
		`exception.threads`,
		`toString(app_exit.reason)`,
		`toString(app_exit.importance)`,
		`app_exit.trace`,
		`app_exit.process_name`,
		`app_exit.pid`,
		`toString(string.severity_text)`,
		`string.string`,
		`toString(gesture_long_click.target)`,
		`toString(gesture_long_click.target_id)`,
		`gesture_long_click.touch_down_time`,
		`gesture_long_click.touch_up_time`,
		`gesture_long_click.width`,
		`gesture_long_click.height`,
		`gesture_long_click.x`,
		`gesture_long_click.y`,
		`toString(gesture_click.target)`,
		`toString(gesture_click.target_id)`,
		`gesture_click.touch_down_time`,
		`gesture_click.touch_up_time`,
		`gesture_click.width`,
		`gesture_click.height`,
		`gesture_click.x`,
		`gesture_click.y`,
		`toString(gesture_scroll.target)`,
		`toString(gesture_scroll.target_id)`,
		`gesture_scroll.touch_down_time`,
		`gesture_scroll.touch_up_time`,
		`gesture_scroll.x`,
		`gesture_scroll.y`,
		`gesture_scroll.end_x`,
		`gesture_scroll.end_y`,
		`toString(gesture_scroll.direction)`,
		`toString(lifecycle_activity.type)`,
		`toString(lifecycle_activity.class_name)`,
		`lifecycle_activity.intent`,
		`lifecycle_activity.saved_instance_state`,
		`toString(lifecycle_fragment.type)`,
		`toString(lifecycle_fragment.class_name)`,
		`lifecycle_fragment.parent_activity`,
		`lifecycle_fragment.tag`,
		`toString(lifecycle_app.type)`,
		`cold_launch.process_start_uptime`,
		`cold_launch.process_start_requested_uptime`,
		`cold_launch.content_provider_attach_uptime`,
		`cold_launch.on_next_draw_uptime`,
		`toString(cold_launch.launched_activity)`,
		`cold_launch.has_saved_state`,
		`cold_launch.intent_data`,
		`cold_launch.duration`,
		`warm_launch.app_visible_uptime`,
		`warm_launch.on_next_draw_uptime`,
		`warm_launch.launched_activity`,
		`warm_launch.has_saved_state`,
		`warm_launch.intent_data`,
		`warm_launch.duration`,
		`hot_launch.app_visible_uptime`,
		`hot_launch.on_next_draw_uptime`,
		`toString(hot_launch.launched_activity)`,
		`hot_launch.has_saved_state`,
		`hot_launch.intent_data`,
		`hot_launch.duration`,
		`toString(network_change.network_type)`,
		`toString(network_change.previous_network_type)`,
		`toString(network_change.network_generation)`,
		`toString(network_change.previous_network_generation)`,
		`toString(network_change.network_provider)`,
		`http.url`,
		`toString(http.method)`,
		`http.status_code`,
		`http.start_time`,
		`http.end_time`,
		`http_request_headers`,
		`http_response_headers`,
		`http.request_body`,
		`http.response_body`,
		`http.failure_reason`,
		`http.failure_description`,
		`toString(http.client)`,
		`memory_usage.java_max_heap`,
		`memory_usage.java_total_heap`,
		`memory_usage.java_free_heap`,
		`memory_usage.total_pss`,
		`memory_usage.rss`,
		`memory_usage.native_total_heap`,
		`memory_usage.native_free_heap`,
		`memory_usage.interval_config`,
		`low_memory.java_max_heap`,
		`low_memory.java_total_heap`,
		`low_memory.java_free_heap`,
		`low_memory.total_pss`,
		`low_memory.rss`,
		`low_memory.native_total_heap`,
		`low_memory.native_free_heap`,
		`toString(trim_memory.level)`,
		`cpu_usage.num_cores`,
		`cpu_usage.clock_speed`,
		`cpu_usage.start_time`,
		`cpu_usage.uptime`,
		`cpu_usage.utime`,
		`cpu_usage.cutime`,
		`cpu_usage.stime`,
		`cpu_usage.cstime`,
		`cpu_usage.interval_config`,
		`toString(navigation.route)`,
	}

	stmt := sqlf.From("default.events")
	defer stmt.Close()

	for i := range cols {
		stmt.Select(cols[i])
	}

	stmt.Where("app_id = ? and session_id = ?", a.ID, sessionId)
	stmt.OrderBy("timestamp")

	rows, err := server.Server.ChPool.Query(ctx, stmt.String(), stmt.Args()...)

	if err != nil {
		return nil, err
	}

	var session Session

	for rows.Next() {
		var ev event.EventField
		var anr event.ANR
		var exception event.Exception
		var exceptionExceptions string
		var exceptionThreads string
		var anrExceptions string
		var anrThreads string
		var appExit event.AppExit
		var logString event.LogString
		var gestureLongClick event.GestureLongClick
		var gestureClick event.GestureClick
		var gestureScroll event.GestureScroll
		var lifecycleActivity event.LifecycleActivity
		var lifecycleFragment event.LifecycleFragment
		var lifecycleApp event.LifecycleApp
		var coldLaunch event.ColdLaunch
		var warmLaunch event.WarmLaunch
		var hotLaunch event.HotLaunch
		var networkChange event.NetworkChange
		var http event.Http
		var memoryUsage event.MemoryUsage
		var lowMemory event.LowMemory
		var trimMemory event.TrimMemory
		var cpuUsage event.CPUUsage
		var navigation event.Navigation

		var coldLaunchDuration uint32
		var warmLaunchDuration uint32
		var hotLaunchDuration uint32

		dest := []any{
			&ev.ID,
			&ev.Type,
			&session.SessionID,
			&session.AppID,
			&ev.IPv4,
			&ev.IPv6,
			&ev.CountryCode,
			&ev.Timestamp,

			// attribute
			&ev.Attribute.InstallationID,
			&ev.Attribute.AppVersion,
			&ev.Attribute.AppBuild,
			&ev.Attribute.AppUniqueID,
			&ev.Attribute.Platform,
			&ev.Attribute.MeasureSDKVersion,
			&ev.Attribute.ThreadName,
			&ev.Attribute.UserID,
			&ev.Attribute.DeviceName,
			&ev.Attribute.DeviceModel,
			&ev.Attribute.DeviceManufacturer,
			&ev.Attribute.DeviceType,
			&ev.Attribute.DeviceIsFoldable,
			&ev.Attribute.DeviceIsPhysical,
			&ev.Attribute.DeviceDensityDPI,
			&ev.Attribute.DeviceWidthPX,
			&ev.Attribute.DeviceHeightPX,
			&ev.Attribute.DeviceDensity,
			&ev.Attribute.DeviceLocale,
			&ev.Attribute.OSName,
			&ev.Attribute.OSVersion,
			&ev.Attribute.NetworkType,
			&ev.Attribute.NetworkGeneration,
			&ev.Attribute.NetworkProvider,

			// anr
			&anr.Fingerprint,
			&anr.Foreground,
			&anrExceptions,
			&anrThreads,

			// excpetion
			&exception.Handled,
			&exception.Fingerprint,
			&exception.Foreground,
			&exceptionExceptions,
			&exceptionThreads,

			// app exit
			&appExit.Reason,
			&appExit.Importance,
			&appExit.Trace,
			&appExit.ProcessName,
			&appExit.PID,

			// log string
			&logString.SeverityText,
			&logString.String,

			// gesture long click
			&gestureLongClick.Target,
			&gestureLongClick.TargetID,
			&gestureLongClick.TouchDownTime,
			&gestureLongClick.TouchUpTime,
			&gestureLongClick.Width,
			&gestureLongClick.Height,
			&gestureLongClick.X,
			&gestureLongClick.Y,

			// gesture click
			&gestureClick.Target,
			&gestureClick.TargetID,
			&gestureClick.TouchDownTime,
			&gestureClick.TouchUpTime,
			&gestureClick.Width,
			&gestureClick.Height,
			&gestureClick.X,
			&gestureClick.Y,

			// gesture scroll
			&gestureScroll.Target,
			&gestureScroll.TargetID,
			&gestureScroll.TouchDownTime,
			&gestureScroll.TouchUpTime,
			&gestureScroll.X,
			&gestureScroll.Y,
			&gestureScroll.EndX,
			&gestureScroll.EndY,
			&gestureScroll.Direction,

			// lifecycle activity
			&lifecycleActivity.Type,
			&lifecycleActivity.ClassName,
			&lifecycleActivity.Intent,
			&lifecycleActivity.SavedInstanceState,

			// lifecycle fragment
			&lifecycleFragment.Type,
			&lifecycleFragment.ClassName,
			&lifecycleFragment.ParentActivity,
			&lifecycleFragment.Tag,

			// lifecycle app
			&lifecycleApp.Type,

			// cold launch
			&coldLaunch.ProcessStartUptime,
			&coldLaunch.ProcessStartRequestedUptime,
			&coldLaunch.ContentProviderAttachUptime,
			&coldLaunch.OnNextDrawUptime,
			&coldLaunch.LaunchedActivity,
			&coldLaunch.HasSavedState,
			&coldLaunch.IntentData,
			&coldLaunchDuration,

			// warm launch
			&warmLaunch.AppVisibleUptime,
			&warmLaunch.OnNextDrawUptime,
			&warmLaunch.LaunchedActivity,
			&warmLaunch.HasSavedState,
			&warmLaunch.IntentData,
			&warmLaunchDuration,

			// hot launch
			&hotLaunch.AppVisibleUptime,
			&hotLaunch.OnNextDrawUptime,
			&hotLaunch.LaunchedActivity,
			&hotLaunch.HasSavedState,
			&hotLaunch.IntentData,
			&hotLaunchDuration,

			// network change
			&networkChange.NetworkType,
			&networkChange.PreviousNetworkType,
			&networkChange.NetworkGeneration,
			&networkChange.PreviousNetworkGeneration,
			&networkChange.NetworkProvider,

			// http
			&http.URL,
			&http.Method,
			&http.StatusCode,
			&http.StartTime,
			&http.EndTime,
			&http.RequestHeaders,
			&http.ResponseHeaders,
			&http.RequestBody,
			&http.ResponseBody,
			&http.FailureReason,
			&http.FailureDescription,
			&http.Client,

			// memory usage
			&memoryUsage.JavaMaxHeap,
			&memoryUsage.JavaTotalHeap,
			&memoryUsage.JavaFreeHeap,
			&memoryUsage.TotalPSS,
			&memoryUsage.RSS,
			&memoryUsage.NativeTotalHeap,
			&memoryUsage.NativeFreeHeap,
			&memoryUsage.IntervalConfig,

			// low memory
			&lowMemory.JavaMaxHeap,
			&lowMemory.JavaTotalHeap,
			&lowMemory.JavaFreeHeap,
			&lowMemory.TotalPSS,
			&lowMemory.RSS,
			&lowMemory.NativeTotalHeap,
			&lowMemory.NativeFreeHeap,

			// trim memory
			&trimMemory.Level,

			// cpu usage
			&cpuUsage.NumCores,
			&cpuUsage.ClockSpeed,
			&cpuUsage.StartTime,
			&cpuUsage.Uptime,
			&cpuUsage.UTime,
			&cpuUsage.CUTime,
			&cpuUsage.STime,
			&cpuUsage.CSTime,
			&cpuUsage.IntervalConfig,

			// navigation
			&navigation.Route,
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		switch ev.Type {
		case event.TypeANR:
			if err := json.Unmarshal([]byte(anrExceptions), &anr.Exceptions); err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(anrThreads), &anr.Threads); err != nil {
				return nil, err
			}
			ev.ANR = &anr
			session.Events = append(session.Events, ev)
		case event.TypeException:
			if err := json.Unmarshal([]byte(exceptionExceptions), &exception.Exceptions); err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(exceptionThreads), &exception.Threads); err != nil {
				return nil, err
			}
			ev.Exception = &exception
			session.Events = append(session.Events, ev)
		case event.TypeAppExit:
			ev.AppExit = &appExit
			session.Events = append(session.Events, ev)
		case event.TypeString:
			ev.LogString = &logString
			session.Events = append(session.Events, ev)
		case event.TypeGestureLongClick:
			ev.GestureLongClick = &gestureLongClick
			session.Events = append(session.Events, ev)
		case event.TypeGestureClick:
			ev.GestureClick = &gestureClick
			session.Events = append(session.Events, ev)
		case event.TypeGestureScroll:
			ev.GestureScroll = &gestureScroll
			session.Events = append(session.Events, ev)
		case event.TypeLifecycleActivity:
			ev.LifecycleActivity = &lifecycleActivity
			session.Events = append(session.Events, ev)
		case event.TypeLifecycleFragment:
			ev.LifecycleFragment = &lifecycleFragment
			session.Events = append(session.Events, ev)
		case event.TypeLifecycleApp:
			ev.LifecycleApp = &lifecycleApp
			session.Events = append(session.Events, ev)
		case event.TypeColdLaunch:
			ev.ColdLaunch = &coldLaunch
			ev.ColdLaunch.Duration = time.Duration(coldLaunchDuration)
			session.Events = append(session.Events, ev)
		case event.TypeWarmLaunch:
			ev.WarmLaunch = &warmLaunch
			ev.WarmLaunch.Duration = time.Duration(warmLaunchDuration)
			session.Events = append(session.Events, ev)
		case event.TypeHotLaunch:
			ev.HotLaunch = &hotLaunch
			ev.HotLaunch.Duration = time.Duration(hotLaunchDuration)
			session.Events = append(session.Events, ev)
		case event.TypeNetworkChange:
			ev.NetworkChange = &networkChange
			session.Events = append(session.Events, ev)
		case event.TypeHttp:
			ev.Http = &http
			session.Events = append(session.Events, ev)
		case event.TypeMemoryUsage:
			ev.MemoryUsage = &memoryUsage
			session.Events = append(session.Events, ev)
		case event.TypeLowMemory:
			ev.LowMemory = &lowMemory
			session.Events = append(session.Events, ev)
		case event.TypeTrimMemory:
			ev.TrimMemory = &trimMemory
			session.Events = append(session.Events, ev)
		case event.TypeCPUUsage:
			ev.CPUUsage = &cpuUsage
			session.Events = append(session.Events, ev)
		case event.TypeNavigation:
			ev.Navigation = &navigation
			session.Events = append(session.Events, ev)
		default:
			continue
		}
	}

	// attach session's first event attribute
	// as the session's attributes
	if len(session.Events) > 0 {
		attr := session.Events[0].Attribute
		session.Attribute = &attr
	}

	return &session, nil
}

func (a App) getIssues(ctx context.Context, af *filter.AppFilter) (events []event.EventField, err error) {
	eventTypes := []any{event.TypeException, event.TypeANR}

	stmt := sqlf.
		From(`default.events`).
		Select(`id`).
		Select(`toString(type)`).
		Select(`timestamp`).
		Select(`session_id`).
		Select(`exception.fingerprint`).
		Select(`anr.fingerprint`).
		Where(`app_id = ?`, a.ID).
		Where("`attribute.app_version` in ?", af.Versions).
		Where("`attribute.app_build` in ?", af.VersionCodes).
		Where("`timestamp` >= ? and `timestamp` <= ?", af.From, af.To).
		Where(`(type = ? or type = ?)`, eventTypes...).
		OrderBy(`timestamp`)

	defer stmt.Close()

	rows, err := server.Server.ChPool.Query(ctx, stmt.String(), stmt.Args()...)
	if err != nil {
		return
	}

	for rows.Next() {
		var ev event.EventField
		var exception event.Exception
		var anr event.ANR
		ev.Exception = &exception
		ev.ANR = &anr

		dest := []any{
			&ev.ID,
			&ev.Type,
			&ev.Timestamp,
			&ev.SessionID,
			&ev.Exception.Fingerprint,
			&ev.ANR.Fingerprint,
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

// getJourneyEvents queries all relevant lifecycle events involved in forming
// an implicit navigational journey.
func (a App) getJourneyEvents(ctx context.Context, af *filter.AppFilter) (events []event.EventField, err error) {
	whereVals := []any{
		event.TypeLifecycleActivity,
		[]string{
			event.LifecycleActivityTypeCreated,
			event.LifecycleActivityTypeResumed,
		},
		event.TypeLifecycleFragment,
		[]string{
			event.LifecycleFragmentTypeAttached,
			event.LifecycleFragmentTypeResumed,
		},
		event.TypeException,
		false,
		event.TypeANR,
	}

	stmt := sqlf.
		From(`default.events`).
		Select(`id`).
		Select(`toString(type)`).
		Select(`timestamp`).
		Select(`session_id`).
		Select(`toString(lifecycle_activity.type)`).
		Select(`toString(lifecycle_activity.class_name)`).
		Select(`toString(lifecycle_fragment.type)`).
		Select(`toString(lifecycle_fragment.class_name)`).
		Select(`toString(lifecycle_fragment.parent_activity)`).
		Where(`app_id = ?`, a.ID).
		Where("`attribute.app_version` in ?", af.Versions).
		Where("`attribute.app_build` in ?", af.VersionCodes).
		Where("`timestamp` >= ? and `timestamp` <= ?", af.From, af.To).
		Where("((type = ? and `lifecycle_activity.type` in ?) or (type = ? and `lifecycle_fragment.type` in ?) or ((type = ? and `exception.handled` = ?) or type = ?))", whereVals...).
		OrderBy(`timestamp`)

	defer stmt.Close()

	rows, err := server.Server.ChPool.Query(ctx, stmt.String(), stmt.Args()...)
	if err != nil {
		return
	}

	for rows.Next() {
		var ev event.EventField
		var lifecycleActivityType string
		var lifecycleActivityClassName string
		var lifecycleFragmentType string
		var lifecycleFragmentClassName string
		var lifecycleFragmentParentActivity string

		dest := []any{
			&ev.ID,
			&ev.Type,
			&ev.Timestamp,
			&ev.SessionID,
			&lifecycleActivityType,
			&lifecycleActivityClassName,
			&lifecycleFragmentType,
			&lifecycleFragmentClassName,
			&lifecycleFragmentParentActivity,
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		if ev.IsLifecycleActivity() {
			ev.LifecycleActivity = &event.LifecycleActivity{
				Type:      lifecycleActivityType,
				ClassName: lifecycleActivityClassName,
			}
		} else if ev.IsLifecycleFragment() {
			ev.LifecycleFragment = &event.LifecycleFragment{
				Type:           lifecycleFragmentType,
				ClassName:      lifecycleFragmentClassName,
				ParentActivity: lifecycleFragmentParentActivity,
			}
		} else if ev.IsException() {
			ev.Exception = &event.Exception{}
		} else if ev.IsANR() {
			ev.ANR = &event.ANR{}
		}

		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

func NewApp(teamId uuid.UUID) *App {
	now := time.Now()
	id := uuid.New()
	return &App{
		ID:        &id,
		TeamId:    teamId,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (a *App) add() (*APIKey, error) {
	id := uuid.New()
	a.ID = &id
	tx, err := server.Server.PgPool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), "insert into public.apps(id, team_id, app_name, created_at, updated_at) values ($1, $2, $3, $4, $5);", a.ID, a.TeamId, a.AppName, a.CreatedAt, a.UpdatedAt)

	if err != nil {
		return nil, err
	}

	apiKey, err := NewAPIKey(*a.ID)

	if err != nil {
		return nil, err
	}

	if err := apiKey.saveTx(tx); err != nil {
		return nil, err
	}

	alertPref := newAlertPref(*a.ID)

	if err := alertPref.insertTx(tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (a *App) getWithTeam(id uuid.UUID) (*App, error) {
	var appName pgtype.Text
	var uniqueId pgtype.Text
	var platform pgtype.Text
	var firstVersion pgtype.Text
	var onboarded pgtype.Bool
	var onboardedAt pgtype.Timestamptz
	var apiKeyLastSeen pgtype.Timestamptz
	var apiKeyCreatedAt pgtype.Timestamptz
	var createdAt pgtype.Timestamptz
	var updatedAt pgtype.Timestamptz

	apiKey := new(APIKey)

	cols := []string{
		"apps.app_name",
		"apps.unique_identifier",
		"apps.platform",
		"apps.first_version",
		"apps.onboarded",
		"apps.onboarded_at",
		"api_keys.key_prefix",
		"api_keys.key_value",
		"api_keys.checksum",
		"api_keys.last_seen",
		"api_keys.created_at",
		"apps.created_at",
		"apps.updated_at",
	}

	stmt := sqlf.PostgreSQL.
		Select(strings.Join(cols, ",")).
		From("public.apps").
		LeftJoin("public.api_keys", "api_keys.app_id = apps.id").
		Where("apps.id = ? and apps.team_id = ?", nil, nil)

	defer stmt.Close()

	dest := []any{
		&appName,
		&uniqueId,
		&platform,
		&firstVersion,
		&onboarded,
		&onboardedAt,
		&apiKey.keyPrefix,
		&apiKey.keyValue,
		&apiKey.checksum,
		&apiKeyLastSeen,
		&apiKeyCreatedAt,
		&createdAt,
		&updatedAt,
	}

	if err := server.Server.PgPool.QueryRow(context.Background(), stmt.String(), id, a.TeamId).Scan(dest...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if appName.Valid {
		a.AppName = appName.String
	}

	if uniqueId.Valid {
		a.UniqueId = uniqueId.String
	} else {
		a.UniqueId = ""
	}

	if platform.Valid {
		a.Platform = platform.String
	} else {
		a.Platform = ""
	}

	if firstVersion.Valid {
		a.FirstVersion = firstVersion.String
	} else {
		a.FirstVersion = ""
	}

	if onboarded.Valid {
		a.Onboarded = onboarded.Bool
	}

	if onboardedAt.Valid {
		a.OnboardedAt = onboardedAt.Time
	}

	if apiKeyLastSeen.Valid {
		apiKey.lastSeen = apiKeyLastSeen.Time
	}

	if apiKeyCreatedAt.Valid {
		apiKey.createdAt = apiKeyCreatedAt.Time
	}

	if createdAt.Valid {
		a.CreatedAt = createdAt.Time
	}

	if updatedAt.Valid {
		a.UpdatedAt = updatedAt.Time
	}

	a.APIKey = apiKey

	return a, nil
}

func (a *App) getTeam(ctx context.Context) (*Team, error) {
	team := &Team{}

	stmt := sqlf.PostgreSQL.
		Select("team_id").
		From("apps").
		Where("id = ?", nil)
	defer stmt.Close()

	if err := server.Server.PgPool.QueryRow(ctx, stmt.String(), a.ID).Scan(&team.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return team, nil
}

func (a *App) Onboard(ctx context.Context, tx *pgx.Tx, uniqueIdentifier, platform, firstVersion string) error {
	now := time.Now()
	stmt := sqlf.PostgreSQL.Update("public.apps").
		Set("onboarded", true).
		Set("unique_identifier", uniqueIdentifier).
		Set("platform", platform).
		Set("first_version", firstVersion).
		Set("onboarded_at", now).
		Set("updated_at", now).
		Where("id = ?", a.ID)

	defer stmt.Close()

	_, err := (*tx).Exec(ctx, stmt.String(), stmt.Args()...)
	if err != nil {
		return err
	}

	return nil
}

// SelectApp selects app by its id.
func SelectApp(ctx context.Context, id uuid.UUID) (app *App, err error) {
	var onboarded pgtype.Bool
	var uniqueId pgtype.Text
	var platform pgtype.Text
	var firstVersion pgtype.Text

	stmt := sqlf.PostgreSQL.
		Select("id").
		Select("onboarded").
		Select("unique_identifier").
		Select("platform").
		Select("first_version").
		From("public.apps").
		Where("id = ?", id)

	defer stmt.Close()

	if app == nil {
		app = &App{}
	}

	if err := server.Server.PgPool.QueryRow(ctx, stmt.String(), stmt.Args()...).Scan(&app.ID, &onboarded, &uniqueId, &platform, &firstVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if onboarded.Valid {
		app.Onboarded = onboarded.Bool
	} else {
		app.Onboarded = false
	}

	if uniqueId.Valid {
		app.UniqueId = uniqueId.String
	} else {
		app.UniqueId = ""
	}

	if platform.Valid {
		app.Platform = platform.String
	} else {
		app.Platform = ""
	}

	if firstVersion.Valid {
		app.FirstVersion = firstVersion.String
	} else {
		app.FirstVersion = ""
	}

	return
}

func GetAppJourney(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `app id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": msg,
		})
		return
	}
	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	af.Expand()

	msg := "app journey request validation failed"

	if err := af.Validate(); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	if err := af.ValidateVersions(); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   msg,
			"details": err.Error(),
		})
		return
	}

	if !af.HasTimeRange() {
		af.SetDefaultTimeRange()
	}

	app := App{
		ID: &id,
	}

	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	msg = `failed to compute app's journey`
	journeyEvents, err := app.getJourneyEvents(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	issueEvents, err := app.getIssues(ctx, &af)
	if err != nil {
		msg = `failed to compute app's journey`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	var exceptionIds []uuid.UUID
	var anrIds []uuid.UUID
	for i := range issueEvents {
		if issueEvents[i].IsException() {
			exceptionIds = append(exceptionIds, issueEvents[i].ID)
		}
		if issueEvents[i].IsANR() {
			anrIds = append(anrIds, issueEvents[i].ID)
		}
	}

	journeyAndroid := journey.NewJourneyAndroid(journeyEvents)

	if err := journeyAndroid.SetNodeExceptionGroups(func(eventIds []uuid.UUID) ([]group.ExceptionGroup, error) {
		exceptionGroups, err := group.GetExceptionGroupsFromExceptionIds(ctx, eventIds)
		if err != nil {
			return nil, err
		}
		return exceptionGroups, nil
	}); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	if err := journeyAndroid.SetNodeANRGroups(func(eventIds []uuid.UUID) ([]group.ANRGroup, error) {
		anrGroups, err := group.GetANRGroupsFromANRIds(ctx, eventIds)
		if err != nil {
			return nil, err
		}
		return anrGroups, nil
	}); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	type Link struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Value  int    `json:"value"`
	}

	type Issue struct {
		ID    uuid.UUID `json:"id"`
		Title string    `json:"title"`
		Count int       `json:"count"`
	}

	type Node struct {
		ID     string `json:"id"`
		Issues gin.H  `json:"issues"`
	}

	var nodes []Node
	var links []Link

	for v := range journeyAndroid.Graph.Order() {
		journeyAndroid.Graph.Visit(v, func(w int, c int64) bool {
			var link Link
			link.Source = journeyAndroid.GetNodeName(v)
			link.Target = journeyAndroid.GetNodeName(w)
			link.Value = journeyAndroid.GetEdgeSessionCount(v, w)
			links = append(links, link)
			return false
		})
	}

	for _, v := range journeyAndroid.GetNodeVertices() {
		var node Node
		name := journeyAndroid.GetNodeName(v)
		exceptionGroups := journeyAndroid.GetNodeExceptionGroups(name)
		crashes := []Issue{}

		for i := range exceptionGroups {
			issue := Issue{
				ID:    exceptionGroups[i].ID,
				Title: exceptionGroups[i].Name,
				Count: exceptionGroups[i].GetMatchingEventCount(exceptionIds),
			}
			crashes = append(crashes, issue)
		}

		sort.Slice(crashes, func(i, j int) bool {
			return crashes[i].Count > crashes[j].Count
		})

		anrGroups := journeyAndroid.GetNodeANRGroups(name)
		anrs := []Issue{}

		for i := range anrGroups {
			issue := Issue{
				ID:    anrGroups[i].ID,
				Title: anrGroups[i].Name,
				Count: anrGroups[i].GetMatchingEventCount(anrIds),
			}
			anrs = append(anrs, issue)
		}

		sort.Slice(anrs, func(i, j int) bool {
			return anrs[i].Count > anrs[j].Count
		})

		node.ID = name
		node.Issues = gin.H{
			"crashes": crashes,
			"anrs":    anrs,
		}
		nodes = append(nodes, node)
	}

	c.JSON(http.StatusOK, gin.H{
		"totalIssues": len(issueEvents),
		"nodes":       nodes,
		"links":       links,
	})
}

func GetAppMetrics(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `app id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": msg,
		})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse app metrics request`
		fmt.Println(msg, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   msg,
			"details": err.Error(),
		})
		return
	}

	af.Expand()

	msg := `app metrics request validation failed`

	if err := af.Validate(); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   msg,
			"details": err.Error(),
		})
		return
	}

	if err := af.ValidateVersions(); err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   msg,
			"details": err.Error(),
		})
		return
	}

	if !af.HasTimeRange() {
		af.SetDefaultTimeRange()
	}

	app := App{
		ID: &id,
	}

	msg = `failed to fetch app metrics`

	launch, err := app.GetLaunchMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	adoption, err := app.GetAdoptionMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	sizes, err := app.GetSizeMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	crashFree, err := app.GetCrashFreeMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	anrFree, err := app.GetANRFreeMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	perceivedCrashFree, err := app.GetPerceivedCrashFreeMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	perceivedANRFree, err := app.GetPerceivedANRFreeMetrics(ctx, &af)
	if err != nil {
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cold_launch": gin.H{
			"p95":   launch.ColdLaunchP95,
			"delta": launch.ColdDelta,
			"nan":   launch.ColdNaN,
		},
		"warm_launch": gin.H{
			"p95":   launch.WarmLaunchP95,
			"delta": launch.WarmDelta,
			"nan":   launch.WarmNaN,
		},
		"hot_launch": gin.H{
			"p95":   launch.HotLaunchP95,
			"delta": launch.HotDelta,
			"nan":   launch.HotNaN,
		},
		"adoption":                      adoption,
		"sizes":                         sizes,
		"crash_free_sessions":           crashFree,
		"anr_free_sessions":             anrFree,
		"perceived_crash_free_sessions": perceivedCrashFree,
		"perceived_anr_free_sessions":   perceivedANRFree,
	})
}

func GetAppFilters(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	ctx := c.Request.Context()

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse query parameters`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	if err := af.Validate(); err != nil {
		msg := "app filters request validation failed"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	app := App{
		ID: &id,
	}

	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	var fl filter.FilterList

	if err := af.GetGenericFilters(ctx, &fl); err != nil {
		msg := `failed to query app filters`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	// club version names & version codes
	var versions []any
	for i := range fl.Versions {
		version := gin.H{"name": fl.Versions[i], "code": fl.VersionCodes[i]}
		versions = append(versions, version)
	}

	c.JSON(http.StatusOK, gin.H{
		"versions":             versions,
		"countries":            fl.Countries,
		"network_providers":    fl.NetworkProviders,
		"network_types":        fl.NetworkTypes,
		"network_generations":  fl.NetworkGenerations,
		"locales":              fl.DeviceLocales,
		"device_manufacturers": fl.DeviceManufacturers,
		"device_names":         fl.DeviceNames,
	})
}

func GetCrashGroups(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse query parameters`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	af.Expand()

	if err := af.Validate(); err != nil {
		msg := "app filters request validation failed"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	if !af.HasTimeRange() {
		af.SetDefaultTimeRange()
	}

	app := App{
		ID: &id,
	}
	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	groups, err := app.GetExceptionGroups(ctx, &af)
	if err != nil {
		msg := "failed to get app's exception groups"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	var crashGroups []group.ExceptionGroup
	for i := range groups {
		ids, err := GetEventIdsMatchingFilter(ctx, groups[i].EventIDs, &af)
		if err != nil {
			msg := "failed to get app's exception group's event ids"
			fmt.Println(msg, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		count := len(ids)

		// only consider those groups that have at least 1 exception
		// event
		if count > 0 {
			groups[i].Count = count

			// omit `event_ids` & `exception_events` fields from JSON
			// response, because these can get really huge
			groups[i].EventIDs = nil
			groups[i].EventExceptions = nil

			crashGroups = append(crashGroups, groups[i])
		}
	}

	group.ComputeCrashContribution(crashGroups)
	group.SortExceptionGroups(crashGroups)
	crashGroups, next, previous := group.PaginateGroups(crashGroups, &af)
	meta := gin.H{"next": next, "previous": previous}

	c.JSON(http.StatusOK, gin.H{
		"results": crashGroups,
		"meta":    meta,
	})
}

func GetCrashGroupCrashes(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	crashGroupId, err := uuid.Parse(c.Param("crashGroupId"))
	if err != nil {
		msg := `crash group id is invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse query parameters`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	af.Expand()

	if err := af.Validate(); err != nil {
		msg := "app filters request validation failed"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	app := App{
		ID: &id,
	}
	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	group, err := app.GetExceptionGroup(ctx, crashGroupId)
	if err != nil {
		msg := fmt.Sprintf("failed to get exception group with id %q", crashGroupId.String())
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	eventExceptions, next, previous, err := GetExceptionsWithFilter(ctx, group.EventIDs, &af)
	if err != nil {
		msg := `failed to get exception group's exception events`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": eventExceptions,
		"meta": gin.H{
			"next":     next,
			"previous": previous,
		},
	})
}

func GetANRGroups(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse query parameters`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	if err := af.Validate(); err != nil {
		msg := "app filters request validation failed"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	if !af.HasTimeRange() {
		af.SetDefaultTimeRange()
	}

	app := App{
		ID: &id,
	}
	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	groups, err := app.GetANRGroups(&af)
	if err != nil {
		msg := "failed to get app's anr groups"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	var anrGroups []group.ANRGroup
	for i := range groups {
		ids, err := GetEventIdsMatchingFilter(ctx, groups[i].EventIDs, &af)
		if err != nil {
			msg := "failed to get app's anr group's event ids"
			fmt.Println(msg, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		count := len(ids)

		// only consider those groups that have at least 1 anr
		// event
		if count > 0 {
			groups[i].Count = count

			// omit `event_ids` & `exception_anrs` fields from JSON
			// response, because these can get really huge
			groups[i].EventIDs = nil
			groups[i].EventANRs = nil

			anrGroups = append(anrGroups, groups[i])
		}
	}

	group.ComputeANRContribution(anrGroups)
	group.SortANRGroups(anrGroups)
	anrGroups, next, previous := group.PaginateGroups(anrGroups, &af)
	meta := gin.H{"next": next, "previous": previous}

	c.JSON(http.StatusOK, gin.H{"results": anrGroups, "meta": meta})
}

func GetANRGroupANRs(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	anrGroupId, err := uuid.Parse(c.Param("anrGroupId"))
	if err != nil {
		msg := `anr group id is invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	af := filter.AppFilter{
		AppID: id,
		Limit: filter.DefaultPaginationLimit,
	}

	if err := c.ShouldBindQuery(&af); err != nil {
		msg := `failed to parse query parameters`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	af.Expand()

	if err := af.Validate(); err != nil {
		msg := "app filters request validation failed"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
		return
	}

	app := App{
		ID: &id,
	}
	team, err := app.getTeam(ctx)
	if err != nil {
		msg := "failed to get team from app id"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf("no team exists for app [%s]", app.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")
	okTeam, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	okApp, err := PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `failed to perform authorization`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if !okTeam || !okApp {
		msg := `you are not authorized to access this app`
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	group, err := app.GetANRGroup(ctx, anrGroupId)
	if err != nil {
		msg := fmt.Sprintf("failed to get anr group with id %q", anrGroupId.String())
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	eventANRs, next, previous, err := GetANRsWithFilter(ctx, group.EventIDs, &af)
	if err != nil {
		msg := `failed to get anr group's anr events`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": eventANRs,
		"meta": gin.H{
			"next":     next,
			"previous": previous,
		},
	})
}

func CreateApp(c *gin.Context) {
	userId := c.GetString("userId")
	teamId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `team id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	ok, err := PerformAuthz(userId, teamId.String(), *ScopeAppAll)
	if err != nil {
		msg := `couldn't perform authorization checks`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if !ok {
		msg := fmt.Sprintf(`you don't have permissions to create apps in team [%s]`, teamId)
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	app := NewApp(teamId)
	if err := c.ShouldBindJSON(&app); err != nil {
		msg := `failed to parse app json payload`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	apiKey, err := app.add()

	if err != nil {
		msg := "failed to create app"
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	app.APIKey = apiKey

	c.JSON(http.StatusCreated, app)
}

func GetAppSession(c *gin.Context) {
	ctx := c.Request.Context()
	appId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `app id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	sessionId, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		msg := `session id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	app := &App{
		ID: &appId,
	}
	team, err := app.getTeam(ctx)
	if err != nil {
		msg := `failed to fetch team from app`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if team == nil {
		msg := fmt.Sprintf(`no team exists for app id: %q`, app.ID)
		fmt.Println(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	userId := c.GetString("userId")

	ok, err := PerformAuthz(userId, team.ID.String(), *ScopeTeamRead)
	if err != nil {
		msg := `couldn't perform authorization checks`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if !ok {
		msg := fmt.Sprintf(`you don't have permissions to read apps in team %q`, team.ID)
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	ok, err = PerformAuthz(userId, team.ID.String(), *ScopeAppRead)
	if err != nil {
		msg := `couldn't perform authorization checks`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if !ok {
		msg := fmt.Sprintf(`you don't have permissions to read apps in team %q`, team.ID)
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	session, err := app.GetSessionEvents(ctx, sessionId)
	if err != nil {
		msg := `failed to fetch session data for replay`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	if len(session.Events) < 1 {
		msg := fmt.Sprintf(`session %q for app %q does not exist`, sessionId, app.ID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": msg,
		})
		return
	}

	duration := session.Duration().Milliseconds()
	cpuUsageEvents := session.EventsOfType(event.TypeCPUUsage)
	cpuUsages := replay.ComputeCPUUsage(cpuUsageEvents)

	memoryUsageEvents := session.EventsOfType(event.TypeMemoryUsage)
	memoryUsages := replay.ComputeMemoryUsage(memoryUsageEvents)

	typeList := []string{
		event.TypeGestureClick,
		event.TypeGestureLongClick,
		event.TypeGestureScroll,
		event.TypeNavigation,
		event.TypeString,
		event.TypeNetworkChange,
		event.TypeColdLaunch,
		event.TypeWarmLaunch,
		event.TypeHotLaunch,
		event.TypeLifecycleActivity,
		event.TypeLifecycleFragment,
		event.TypeLifecycleApp,
		event.TypeTrimMemory,
		event.TypeLowMemory,
		event.TypeAppExit,
		event.TypeException,
		event.TypeANR,
		event.TypeHttp,
	}

	eventMap := session.EventsOfTypes(typeList...)
	threads := make(replay.Threads)

	gestureClickEvents := eventMap[event.TypeGestureClick]
	if len(gestureClickEvents) > 0 {
		gestureClicks := replay.ComputeGestureClicks(gestureClickEvents)
		threadedGestureClicks := replay.GroupByThreads(gestureClicks)
		threads.Organize(event.TypeGestureClick, threadedGestureClicks)
	}

	gestureLongClickEvents := eventMap[event.TypeGestureLongClick]
	if len(gestureLongClickEvents) > 0 {
		gestureLongClicks := replay.ComputeGestureLongClicks(gestureLongClickEvents)
		threadedGestureLongClicks := replay.GroupByThreads(gestureLongClicks)
		threads.Organize(event.TypeGestureLongClick, threadedGestureLongClicks)
	}

	gestureScrollEvents := eventMap[event.TypeGestureScroll]
	if len(gestureScrollEvents) > 0 {
		gestureScrolls := replay.ComputeGestureScrolls(gestureScrollEvents)
		threadedGestureScrolls := replay.GroupByThreads(gestureScrolls)
		threads.Organize(event.TypeGestureScroll, threadedGestureScrolls)
	}

	navEvents := eventMap[event.TypeNavigation]
	if len(navEvents) > 0 {
		navs := replay.ComputeNavigation(navEvents)
		threadedNavs := replay.GroupByThreads(navs)
		threads.Organize(event.TypeNavigation, threadedNavs)
	}

	logEvents := eventMap[event.TypeString]
	if len(logEvents) > 0 {
		logs := replay.ComputeLogString(logEvents)
		threadedLogs := replay.GroupByThreads(logs)
		threads.Organize(event.TypeString, threadedLogs)
	}

	netChangeEvents := eventMap[event.TypeNetworkChange]
	if len(netChangeEvents) > 0 {
		netChanges := replay.ComputeNetworkChange(netChangeEvents)
		threadedNetChanges := replay.GroupByThreads(netChanges)
		threads.Organize(event.TypeNetworkChange, threadedNetChanges)
	}

	coldLaunchEvents := eventMap[event.TypeColdLaunch]
	if len(coldLaunchEvents) > 0 {
		coldLaunches := replay.ComputeColdLaunches(coldLaunchEvents)
		threadedColdLaunches := replay.GroupByThreads(coldLaunches)
		threads.Organize(event.TypeColdLaunch, threadedColdLaunches)
	}

	warmLaunchEvents := eventMap[event.TypeWarmLaunch]
	if len(warmLaunchEvents) > 0 {
		warmLaunches := replay.ComputeWarmLaunches(warmLaunchEvents)
		threadedWarmLaunches := replay.GroupByThreads(warmLaunches)
		threads.Organize(event.TypeWarmLaunch, threadedWarmLaunches)
	}

	hotLaunchEvents := eventMap[event.TypeHotLaunch]
	if len(hotLaunchEvents) > 0 {
		hotLaunches := replay.ComputeHotLaunches(hotLaunchEvents)
		threadedHotLaunches := replay.GroupByThreads(hotLaunches)
		threads.Organize(event.TypeHotLaunch, threadedHotLaunches)
	}

	lifecycleActivityEvents := eventMap[event.TypeLifecycleActivity]
	if len(lifecycleActivityEvents) > 0 {
		lifecycleActivities := replay.ComputeLifecycleActivities(lifecycleActivityEvents)
		threadedLifecycleActivities := replay.GroupByThreads(lifecycleActivities)
		threads.Organize(event.TypeLifecycleActivity, threadedLifecycleActivities)
	}

	lifecycleFragmentEvents := eventMap[event.TypeLifecycleFragment]
	if len(lifecycleActivityEvents) > 0 {
		lifecycleFragments := replay.ComputeLifecycleFragments(lifecycleFragmentEvents)
		threadedLifecycleFragments := replay.GroupByThreads(lifecycleFragments)
		threads.Organize(event.TypeLifecycleFragment, threadedLifecycleFragments)
	}

	lifecycleAppEvents := eventMap[event.TypeLifecycleApp]
	if len(lifecycleActivityEvents) > 0 {
		lifecycleApps := replay.ComputeLifecycleApps(lifecycleAppEvents)
		threadedLifecycleApps := replay.GroupByThreads(lifecycleApps)
		threads.Organize(event.TypeLifecycleApp, threadedLifecycleApps)
	}

	trimMemoryEvents := eventMap[event.TypeTrimMemory]
	if len(trimMemoryEvents) > 0 {
		trimMemories := replay.ComputeTrimMemories(trimMemoryEvents)
		threadedTrimMemories := replay.GroupByThreads(trimMemories)
		threads.Organize(event.TypeTrimMemory, threadedTrimMemories)
	}

	lowMemoryEvents := eventMap[event.TypeLowMemory]
	if len(lowMemoryEvents) > 0 {
		lowMemories := replay.ComputeLowMemories(lowMemoryEvents)
		threadedLowMemories := replay.GroupByThreads(lowMemories)
		threads.Organize(event.TypeLowMemory, threadedLowMemories)
	}

	appExitEvents := eventMap[event.TypeAppExit]
	if len(appExitEvents) > 0 {
		appExits := replay.ComputeAppExits(appExitEvents)
		threadedAppExits := replay.GroupByThreads(appExits)
		threads.Organize(event.TypeAppExit, threadedAppExits)
	}

	exceptionEvents := eventMap[event.TypeException]
	if len(exceptionEvents) > 0 {
		exceptions := replay.ComputeExceptions(exceptionEvents)
		threadedExceptions := replay.GroupByThreads(exceptions)
		threads.Organize(event.TypeException, threadedExceptions)
	}

	anrEvents := eventMap[event.TypeANR]
	if len(anrEvents) > 0 {
		anrs := replay.ComputeANRs(anrEvents)
		threadedANRs := replay.GroupByThreads(anrs)
		threads.Organize(event.TypeANR, threadedANRs)
	}

	httpEvents := eventMap[event.TypeHttp]
	if len(httpEvents) > 0 {
		httpies := replay.ComputeHttp(httpEvents)
		threadedHttpies := replay.GroupByThreads(httpies)
		threads.Organize(event.TypeHttp, threadedHttpies)
	}

	threads.Sort()

	response := gin.H{
		"session_id":   sessionId,
		"attribute":    session.Attribute,
		"app_id":       appId,
		"duration":     duration,
		"cpu_usage":    cpuUsages,
		"memory_usage": memoryUsages,
		"threads":      threads,
	}

	c.JSON(http.StatusOK, response)
}

func GetAlertPrefs(c *gin.Context) {
	appId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := `app id invalid or missing`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	alertPref, err := getAlertPref(appId)
	if err != nil {
		msg := `unable to fetch notif prefs`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, alertPref)
}

func UpdateAlertPrefs(c *gin.Context) {
	ctx := c.Request.Context()
	userId := c.GetString("userId")

	appId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		msg := "app id invalid or missing"
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	app := &App{
		ID: &appId,
	}

	var team *Team
	if team, err = app.getTeam(ctx); err != nil {
		msg := "failed to get app"
		fmt.Println(msg, err)
		return
	}

	ok, err := PerformAuthz(userId, team.ID.String(), *ScopeAppAll)
	if err != nil {
		msg := `couldn't perform authorization checks`
		fmt.Println(msg, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}
	if !ok {
		msg := fmt.Sprintf(`you don't have permissions to update alert preferences in team [%s]`, app.TeamId)
		c.JSON(http.StatusForbidden, gin.H{"error": msg})
		return
	}

	alertPref := newAlertPref(appId)

	var payload AlertPrefPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		msg := `failed to parse alert preferences json payload`
		fmt.Println(msg, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	alertPref.CrashRateSpikeEmail = payload.CrashRateSpike.Email
	alertPref.CrashRateSpikeSlack = payload.CrashRateSpike.Slack
	alertPref.AnrRateSpikeEmail = payload.AnrRateSpike.Email
	alertPref.AnrRateSpikeSlack = payload.AnrRateSpike.Slack
	alertPref.LaunchTimeSpikeEmail = payload.LaunchTimeSpike.Email
	alertPref.LaunchTimeSpikeSlack = payload.LaunchTimeSpike.Slack

	alertPref.update()

	c.JSON(http.StatusOK, gin.H{"ok": "done"})
}
