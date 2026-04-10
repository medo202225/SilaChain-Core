package consensus

// CANONICAL OWNERSHIP: root consensus package is limited to beacon state, scheduling, transition coordination, and validator coordination.
// Engine, engine API, forkchoice, runtime, txpool, executionstate, and p2p ownership live in their dedicated subpackages.

import (
	"context"
	"log"
	"time"

	validatorclient "silachain/internal/validatorclient"
)

type ProposalExecutor interface {
	ProposeBlock(ctx context.Context) error
}

type Scheduler struct {
	validatorPublicKey string
	signer             validatorclient.DutySigner
	dutyProvider       DutyProvider
	proposalExecutor   ProposalExecutor
	forkchoiceNotifier SlotAwareForkchoiceNotifier
	coordinator        *TransitionCoordinator
	slotDuration       time.Duration
	slotsPerEpoch      uint64
	state              *BeaconStateV1
}

func NewScheduler(
	validatorPublicKey string,
	signer validatorclient.DutySigner,
	dutyProvider DutyProvider,
	proposalExecutor ProposalExecutor,
	forkchoiceNotifier SlotAwareForkchoiceNotifier,
	slotDuration time.Duration,
	slotsPerEpoch uint64,
	state *BeaconStateV1,
) *Scheduler {
	if slotDuration <= 0 {
		slotDuration = 12 * time.Second
	}
	if slotsPerEpoch == 0 {
		slotsPerEpoch = 32
	}
	if state == nil {
		state = NewBeaconStateV1(nil)
		state.AdvanceSlot(0, slotsPerEpoch)
	}

	return &Scheduler{
		validatorPublicKey: validatorPublicKey,
		signer:             signer,
		dutyProvider:       dutyProvider,
		proposalExecutor:   proposalExecutor,
		forkchoiceNotifier: forkchoiceNotifier,
		coordinator:        NewTransitionCoordinator(forkchoiceNotifier, proposalExecutor),
		slotDuration:       slotDuration,
		slotsPerEpoch:      slotsPerEpoch,
		state:              state,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.slotDuration)
	defer ticker.Stop()

	var slot uint64

	for {
		select {
		case <-ctx.Done():
			log.Printf("consensus scheduler stopped")
			return
		case <-ticker.C:
			s.state.AdvanceSlot(slot, s.slotsPerEpoch)

			if provider, ok := s.dutyProvider.(SlotAwareDutyProvider); ok {
				provider.AdvanceToSlot(slot)
			}

			s.runSlot(ctx)
			slot++
		}
	}
}

func (s *Scheduler) runSlot(ctx context.Context) {
	if s.signer == nil {
		log.Printf("consensus scheduler: duty signer is nil")
		return
	}
	if s.dutyProvider == nil {
		log.Printf("consensus scheduler: duty provider is nil")
		return
	}
	if s.coordinator == nil {
		log.Printf("consensus scheduler: transition coordinator is nil")
		return
	}

	active := s.state.ActiveValidators()
	exited := s.state.ExitedValidators()
	slashed := s.state.SlashedValidators()
	proposer := s.state.ProposerForCurrentSlot()
	attesters := s.state.AttestersForCurrentSlot()

	selectedProposer := ""
	if proposer != nil {
		selectedProposer = proposer.PublicKey
	}

	localAttester := s.state.AttesterByPublicKey(s.validatorPublicKey)
	selectedAttester := ""
	if localAttester != nil {
		selectedAttester = localAttester.PublicKey
	}

	log.Printf(
		"consensus scheduler: slot=%d slot_index_in_epoch=%d epoch=%d committee_count=%d committee_index=%d justified=(epoch=%d root=%s) finalized=(epoch=%d root=%s) active_validators=%d active_balance=%d exited_validators=%d slashed_validators=%d attesters=%d selected_proposer=%s local_attester=%s local_validator=%s",
		s.state.Slot,
		s.state.slotIndexInEpoch(s.slotsPerEpoch),
		s.state.Epoch,
		s.state.CommitteeCount(s.slotsPerEpoch),
		s.state.CommitteeIndexForCurrentSlot(s.slotsPerEpoch),
		s.state.CurrentJustifiedCheckpoint.Epoch,
		s.state.CurrentJustifiedCheckpoint.Root,
		s.state.FinalizedCheckpoint.Epoch,
		s.state.FinalizedCheckpoint.Root,
		len(active),
		s.state.TotalActiveBalance(),
		len(exited),
		len(slashed),
		len(attesters),
		selectedProposer,
		selectedAttester,
		s.validatorPublicKey,
	)

	proposalDuty, hasProposalDuty, err := s.dutyProvider.NextProposalDuty(ctx)
	if err != nil {
		log.Printf("consensus scheduler: proposal duty error: %v", err)
		return
	}

	shouldPropose := false
	if hasProposalDuty && proposalDuty != nil {
		if proposalDuty.PublicKey == s.validatorPublicKey {
			sig, err := s.signer.SignProposal(ctx, *proposalDuty)
			if err != nil {
				log.Printf("consensus scheduler: proposal signing failed: %v", err)
				return
			}
			log.Printf("consensus scheduler: proposal duty signed validator=%s slot=%d signature=%s", s.validatorPublicKey, proposalDuty.Slot, sig.SignatureHex)
			shouldPropose = true
		} else {
			log.Printf("consensus scheduler: slot=%d proposer=%s local_validator=%s proposal skipped", proposalDuty.Slot, proposalDuty.PublicKey, s.validatorPublicKey)
		}
	}

	transitionResult, err := s.coordinator.RunForkchoiceAndMaybeProposeWithResult(ctx, s.state, shouldPropose)
	if err != nil {
		log.Printf("consensus scheduler: transition coordination failed: %v", err)
		return
	}

	if transitionResult != nil {
		log.Printf(
			"consensus scheduler: transition result forkchoice_applied=%t proposal_applied=%t payload_id=%s",
			transitionResult.ForkchoiceApplied,
			transitionResult.ProposalApplied,
			transitionResult.PayloadID,
		)
		if transitionResult.ProposalResult != nil {
			log.Printf(
				"consensus scheduler: proposal result payload_status=%s payload_accepted=%t",
				transitionResult.ProposalResult.PayloadStatus,
				transitionResult.ProposalResult.PayloadAccepted,
			)
		}
	}

	if shouldPropose {
		log.Printf("consensus scheduler: proposal executor completed validator=%s slot=%d", s.validatorPublicKey, proposalDuty.Slot)
	}

	attDuty, hasAttDuty, err := s.dutyProvider.NextAttestationDuty(ctx)
	if err != nil {
		log.Printf("consensus scheduler: attestation duty error: %v", err)
		return
	}
	if hasAttDuty && attDuty != nil {
		if s.state.HasAttester(s.validatorPublicKey) {
			attDuty.PublicKey = s.validatorPublicKey
			sig, err := s.signer.SignAttestation(ctx, *attDuty)
			if err != nil {
				log.Printf("consensus scheduler: attestation signing failed: %v", err)
				return
			}
			log.Printf("consensus scheduler: attestation duty signed validator=%s source_epoch=%d target_epoch=%d signature=%s", s.validatorPublicKey, attDuty.SourceEpoch, attDuty.TargetEpoch, sig.SignatureHex)
		} else {
			log.Printf("consensus scheduler: slot=%d attestation skipped local_validator=%s", s.state.Slot, s.validatorPublicKey)
		}
	}
}
