package collection

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-autoupdate-service/internal/restrict/perm"
	"github.com/OpenSlides/openslides-autoupdate-service/pkg/datastore/dsfetch"
)

// Meeting handels restrictions of the collection meeting.
//
// The user can see a meeting if one of the following is True:
//
//	`meeting/enable_anonymous`.
//	The user is in meeting/user_ids.
//	The user has the CML can_manage of the meeting's committee.
//	The user has the CML can_manage of any meeting and the meeting is a template meeting.
//	The user has the OML can_manage_organization.
//
// Mode A: Always visible to everyone.
//
// Mode B: The user can see the meeting.
//
// Mode C: The user has meeting.can_see_frontpage.
//
// Mode D: The user has meeting.can_see_livestream.
type Meeting struct{}

// Name returns the collection name.
func (m Meeting) Name() string {
	return "meeting"
}

// MeetingID returns the meetingID for the object.
func (m Meeting) MeetingID(ctx context.Context, ds *dsfetch.Fetch, id int) (int, bool, error) {
	return id, true, nil
}

// Modes returns the restrictions modes for the meeting collection.
func (m Meeting) Modes(mode string) FieldRestricter {
	switch mode {
	case "A":
		return Allways
	case "B":
		return m.see
	case "C":
		return m.modeC
	case "D":
		return m.modeD
	}
	return nil
}

func (m Meeting) see(ctx context.Context, ds *dsfetch.Fetch, meetingIDs ...int) ([]int, error) {
	requestUser, err := perm.RequestUserFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting request user: %w", err)
	}

	oml, err := perm.HasOrganizationManagementLevel(ctx, ds, requestUser, perm.OMLCanManageOrganization)
	if err != nil {
		return nil, fmt.Errorf("checking organization management level: %w", err)
	}

	if oml {
		return meetingIDs, nil
	}

	return eachCondition(meetingIDs, func(meetingID int) (bool, error) {
		enableAnonymous, err := ds.Meeting_EnableAnonymous(meetingID).Value(ctx)
		if err != nil {
			return false, fmt.Errorf("checking enabled anonymous: %w", err)
		}
		if enableAnonymous {
			return true, nil
		}

		if requestUser == 0 {
			return false, nil
		}

		userIDs, err := ds.Meeting_UserIDs(meetingID).Value(ctx)
		if err != nil {
			return false, fmt.Errorf("getting user ids of meeting: %w", err)
		}
		for _, id := range userIDs {
			if requestUser == id {
				return true, nil
			}
		}

		committeeID, err := ds.Meeting_CommitteeID(meetingID).Value(ctx)
		if err != nil {
			return false, fmt.Errorf("getting committee id of meeting: %w", err)
		}

		isCommitteeManager, err := perm.HasCommitteeManagementLevel(ctx, ds, requestUser, committeeID)
		if err != nil {
			return false, fmt.Errorf("getting committee management status: %w", err)
		}

		if isCommitteeManager {
			return true, nil
		}

		_, isTemplateMeeting, err := ds.Meeting_TemplateForOrganizationID(meetingID).Value(ctx)
		if err != nil {
			return false, fmt.Errorf("getting template meeting: %w", err)
		}

		cmlMeetings, err := perm.ManagementLevelCommittees(ctx, ds, requestUser)
		if err != nil {
			return false, fmt.Errorf("getting meetings with cml can manage: %w", err)
		}

		if isTemplateMeeting && len(cmlMeetings) > 0 {
			return true, nil
		}

		return false, nil
	})
}

func (m Meeting) modeC(ctx context.Context, ds *dsfetch.Fetch, meetingIDs ...int) ([]int, error) {
	allowed, err := eachCondition(meetingIDs, func(meetingID int) (bool, error) {
		perms, err := perm.FromContext(ctx, meetingID)
		if err != nil {
			return false, fmt.Errorf("getting permissions: %w", err)
		}

		return perms.Has(perm.MeetingCanSeeFrontpage), nil
	})
	if err != nil {
		return nil, fmt.Errorf("checking can front page permission: %w", err)
	}

	return allowed, nil
}

func (m Meeting) modeD(ctx context.Context, ds *dsfetch.Fetch, meetingIDs ...int) ([]int, error) {
	allowed, err := eachCondition(meetingIDs, func(meetingID int) (bool, error) {
		perms, err := perm.FromContext(ctx, meetingID)
		if err != nil {
			return false, fmt.Errorf("getting permissions: %w", err)
		}

		return perms.Has(perm.MeetingCanSeeLivestream), nil
	})
	if err != nil {
		return nil, fmt.Errorf("checking can see lievestream permission: %w", err)
	}

	return allowed, nil
}
