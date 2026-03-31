package grpcapi

import (
	"context"
	"errors"
	"net"
	"time"

	persboardv1 "persboard/backend/api/gen/api/proto/persboard/v1"
	"persboard/backend/internal/domain"
	"persboard/backend/internal/service"
	orgusecase "persboard/backend/internal/usecase/org"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type orgUseCase interface {
	Health(ctx context.Context) (map[string]string, error)
	DashboardMetrics(ctx context.Context) (domain.DashboardResponse, error)
	PeopleStats(ctx context.Context) (domain.PersonStats, error)
	OrgStructure(ctx context.Context) (domain.OrgStructureResponse, error)
	CreateTeam(ctx context.Context, input domain.CreateTeamInput) (int, error)
	UpdateTeam(ctx context.Context, id int, input domain.UpdateTeamInput) error
	DeleteTeam(ctx context.Context, id int) error
	CreatePerson(ctx context.Context, input domain.CreatePersonInput) (int, error)
	UpdatePerson(ctx context.Context, id int, input domain.UpdatePersonInput) error
	DeletePerson(ctx context.Context, id int) error
}

type calendarUseCase interface {
	CalendarMetrics(ctx context.Context, from, to time.Time) (domain.CalendarMetricsResponse, error)
	UpdateMetricWeight(ctx context.Context, input domain.UpdateMetricWeightInput) (domain.MetricWeight, error)
}

type Server struct {
	persboardv1.UnimplementedOrgServiceServer
	orgUC      orgUseCase
	calendarUC calendarUseCase
}

func NewServer(orgUC orgUseCase, calendarUC calendarUseCase) *Server {
	return &Server{orgUC: orgUC, calendarUC: calendarUC}
}

func StartGRPCServer(addr string, orgUC orgUseCase, calendarUC calendarUseCase) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	persboardv1.RegisterOrgServiceServer(srv, NewServer(orgUC, calendarUC))
	reflection.Register(srv)
	return srv.Serve(lis)
}

func NewGatewayMux(ctx context.Context, grpcEndpoint string) (*runtime.ServeMux, error) {
	mux := runtime.NewServeMux()
	if err := persboardv1.RegisterOrgServiceHandlerFromEndpoint(
		ctx,
		mux,
		grpcEndpoint,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		return nil, err
	}
	return mux, nil
}

func (s *Server) Health(ctx context.Context, _ *emptypb.Empty) (*persboardv1.HealthResponse, error) {
	resp, err := s.orgUC.Health(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "health check failed")
	}
	return &persboardv1.HealthResponse{
		Status: resp["status"],
		Db:     resp["db"],
	}, nil
}

func (s *Server) GetDashboardMetrics(ctx context.Context, _ *emptypb.Empty) (*persboardv1.DashboardMetricsResponse, error) {
	resp, err := s.orgUC.DashboardMetrics(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch dashboard metrics")
	}

	metrics := make([]*persboardv1.Metric, 0, len(resp.Metrics))
	for _, item := range resp.Metrics {
		metrics = append(metrics, &persboardv1.Metric{
			Key:   item.Key,
			Title: item.Title,
			Value: item.Value,
			Trend: item.Trend,
		})
	}

	return &persboardv1.DashboardMetricsResponse{
		UpdatedAt: timestamppb.New(resp.UpdatedAt),
		Metrics:   metrics,
	}, nil
}

func (s *Server) GetPeopleStats(ctx context.Context, _ *emptypb.Empty) (*persboardv1.PeopleStatsResponse, error) {
	resp, err := s.orgUC.PeopleStats(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch people stats")
	}
	return &persboardv1.PeopleStatsResponse{
		TotalPeople:     int32(resp.TotalPeople),
		ActivePeople:    int32(resp.ActivePeople),
		AverageVelocity: resp.AverageVelocity,
	}, nil
}

func (s *Server) GetOrgStructure(ctx context.Context, _ *emptypb.Empty) (*persboardv1.OrgStructureResponse, error) {
	resp, err := s.orgUC.OrgStructure(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch org structure")
	}

	teams := make([]*persboardv1.Team, 0, len(resp.Teams))
	for _, t := range resp.Teams {
		members := make([]*persboardv1.Person, 0, len(t.Members))
		for _, m := range t.Members {
			person := &persboardv1.Person{
				Id:       int32(m.ID),
				FullName: m.FullName,
				Role:     m.Role,
				Velocity: m.Velocity,
				IsActive: m.IsActive,
				TeamId:   int32(m.TeamID),
			}
			if m.TeamLeadID != nil {
				person.TeamLeadId = protoInt32(int32(*m.TeamLeadID))
			}
			if m.BirthDate != nil {
				person.BirthDate = protoString(*m.BirthDate)
			}
			if m.EmploymentDate != nil {
				person.EmploymentDate = protoString(*m.EmploymentDate)
			}
			members = append(members, person)
		}

		team := &persboardv1.Team{
			Id:      int32(t.ID),
			Name:    t.Name,
			Members: members,
		}
		if t.LeadID != nil {
			team.LeadId = protoInt32(int32(*t.LeadID))
		}
		teams = append(teams, team)
	}

	return &persboardv1.OrgStructureResponse{
		UpdatedAt: timestamppb.New(resp.UpdatedAt),
		Teams:     teams,
	}, nil
}

func (s *Server) CreateTeam(ctx context.Context, req *persboardv1.CreateTeamRequest) (*persboardv1.TeamMutationResponse, error) {
	var leadID *int
	if req.LeadId != nil {
		lead := int(req.GetLeadId())
		leadID = &lead
	}

	id, err := s.orgUC.CreateTeam(ctx, domain.CreateTeamInput{
		Name:   req.GetName(),
		LeadID: leadID,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &persboardv1.TeamMutationResponse{
		Id:   int32(id),
		Name: req.GetName(),
	}, nil
}

func (s *Server) UpdateTeam(ctx context.Context, req *persboardv1.UpdateTeamRequest) (*persboardv1.TeamMutationResponse, error) {
	var leadID *int
	if req.LeadId != nil {
		lead := int(req.GetLeadId())
		leadID = &lead
	}

	id := int(req.GetId())
	err := s.orgUC.UpdateTeam(ctx, id, domain.UpdateTeamInput{
		Name:   req.GetName(),
		LeadID: leadID,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &persboardv1.TeamMutationResponse{
		Id:   int32(id),
		Name: req.GetName(),
	}, nil
}

func (s *Server) DeleteTeam(ctx context.Context, req *persboardv1.DeleteByIDRequest) (*emptypb.Empty, error) {
	if err := s.orgUC.DeleteTeam(ctx, int(req.GetId())); err != nil {
		return nil, toStatusError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CreatePerson(ctx context.Context, req *persboardv1.CreatePersonRequest) (*persboardv1.PersonMutationResponse, error) {
	var teamLeadID *int
	if req.TeamLeadId != nil {
		lead := int(req.GetTeamLeadId())
		teamLeadID = &lead
	}

	id, err := s.orgUC.CreatePerson(ctx, domain.CreatePersonInput{
		FullName:       req.GetFullName(),
		Role:           req.GetRole(),
		Velocity:       req.GetVelocity(),
		IsActive:       req.GetIsActive(),
		TeamID:         int(req.GetTeamId()),
		TeamLeadID:     teamLeadID,
		BirthDate:      optionalString(req.BirthDate),
		EmploymentDate: optionalString(req.EmploymentDate),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &persboardv1.PersonMutationResponse{
		Id:       int32(id),
		FullName: req.GetFullName(),
	}, nil
}

func (s *Server) UpdatePerson(ctx context.Context, req *persboardv1.UpdatePersonRequest) (*persboardv1.PersonMutationResponse, error) {
	var teamLeadID *int
	if req.TeamLeadId != nil {
		lead := int(req.GetTeamLeadId())
		teamLeadID = &lead
	}

	id := int(req.GetId())
	err := s.orgUC.UpdatePerson(ctx, id, domain.UpdatePersonInput{
		FullName:       req.GetFullName(),
		Role:           req.GetRole(),
		Velocity:       req.GetVelocity(),
		IsActive:       req.GetIsActive(),
		TeamID:         int(req.GetTeamId()),
		TeamLeadID:     teamLeadID,
		BirthDate:      optionalString(req.BirthDate),
		EmploymentDate: optionalString(req.EmploymentDate),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &persboardv1.PersonMutationResponse{
		Id:       int32(id),
		FullName: req.GetFullName(),
	}, nil
}

func (s *Server) DeletePerson(ctx context.Context, req *persboardv1.DeleteByIDRequest) (*emptypb.Empty, error) {
	if err := s.orgUC.DeletePerson(ctx, int(req.GetId())); err != nil {
		return nil, toStatusError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetCalendarMetrics(ctx context.Context, req *persboardv1.GetCalendarMetricsRequest) (*persboardv1.CalendarMetricsResponse, error) {
	from, to, err := parseDateRange(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, toStatusError(err)
	}

	resp, err := s.calendarUC.CalendarMetrics(ctx, from, to)
	if err != nil {
		return nil, toStatusError(err)
	}

	metrics := make([]*persboardv1.CalendarMetric, 0, len(resp.Metrics))
	for _, m := range resp.Metrics {
		valuesByDate := make(map[string]*persboardv1.CalendarMetricCellValue, len(m.ValuesByDate))
		for day, value := range m.ValuesByDate {
			cell := &persboardv1.CalendarMetricCellValue{}
			if value != nil {
				if value.Number != nil {
					cell.Number = protoFloat64(*value.Number)
				}
				if value.Bool != nil {
					cell.Bool = protoBool(*value.Bool)
				}
			}
			valuesByDate[day] = cell
		}

		target := &persboardv1.TargetValue{}
		if m.TargetValue.Number != nil {
			target.Number = protoFloat64(*m.TargetValue.Number)
		}
		if m.TargetValue.Bool != nil {
			target.Bool = protoBool(*m.TargetValue.Bool)
		}

		metrics = append(metrics, &persboardv1.CalendarMetric{
			Key:            m.Key,
			Title:          m.Title,
			Weight:         m.Weight,
			MetricType:     string(m.MetricType),
			TargetValue:    target,
			TargetOperator: string(m.TargetOperator),
			ValuesByDate:   valuesByDate,
		})
	}

	return &persboardv1.CalendarMetricsResponse{
		From:    resp.From,
		To:      resp.To,
		Days:    resp.Days,
		Metrics: metrics,
	}, nil
}

func (s *Server) UpdateMetricWeight(ctx context.Context, req *persboardv1.UpdateMetricWeightRequest) (*persboardv1.MetricWeightResponse, error) {
	resp, err := s.calendarUC.UpdateMetricWeight(ctx, domain.UpdateMetricWeightInput{
		MetricKey: req.GetMetricKey(),
		Weight:    req.GetWeight(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &persboardv1.MetricWeightResponse{
		Key:    resp.Key,
		Title:  resp.Title,
		Weight: resp.Weight,
	}, nil
}

func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	parse := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, nil
		}
		return time.Parse("2006-01-02", s)
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -6)
	to := now

	if d, err := parse(fromStr); err != nil {
		return time.Time{}, time.Time{}, orgusecase.ValidationError{Message: "invalid from date"}
	} else if !d.IsZero() {
		from = d
	}
	if d, err := parse(toStr); err != nil {
		return time.Time{}, time.Time{}, orgusecase.ValidationError{Message: "invalid to date"}
	} else if !d.IsZero() {
		to = d
	}

	if from.After(to) {
		return time.Time{}, time.Time{}, orgusecase.ValidationError{Message: "from must be <= to"}
	}

	days := int(to.Sub(from).Hours()/24) + 1
	if days < 1 || days > 31 {
		return time.Time{}, time.Time{}, orgusecase.ValidationError{Message: "date range must be between 1 and 31 days"}
	}

	return from, to, nil
}

func toStatusError(err error) error {
	var validationErr orgusecase.ValidationError
	if errors.As(err, &validationErr) {
		return status.Error(codes.InvalidArgument, validationErr.Message)
	}
	var calendarValidationErr service.ValidationError
	if errors.As(err, &calendarValidationErr) {
		return status.Error(codes.InvalidArgument, calendarValidationErr.Message)
	}
	var notFoundErr orgusecase.NotFoundError
	if errors.As(err, &notFoundErr) {
		return status.Error(codes.NotFound, notFoundErr.Message)
	}
	return status.Error(codes.Internal, "internal server error")
}

func protoInt32(v int32) *int32 {
	return &v
}

func protoFloat64(v float64) *float64 {
	return &v
}

func protoBool(v bool) *bool {
	return &v
}

func protoString(v string) *string {
	return &v
}

func optionalString(value *string) *string {
	if value == nil {
		return nil
	}
	return value
}
