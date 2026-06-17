package service

//go:generate mockgen -destination=mocks/mocks.go -package=mocks github.com/kurt4ins/vk-segmentation/internal/service SegmentRepository,UserRepository,MembershipRepository,HistoryRepository,HistoryReader,RolloutEnqueuer
