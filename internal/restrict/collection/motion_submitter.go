package collection

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-autoupdate-service/internal/restrict/perm"
	"github.com/OpenSlides/openslides-autoupdate-service/pkg/datastore"
)

// MotionSubmitter handels restrictions of the collection motion_submitter.
type MotionSubmitter struct{}

// Modes returns the restrictions modes for the meeting collection.
func (m MotionSubmitter) Modes(mode string) FieldRestricter {
	switch mode {
	case "A":
		return m.see
	}
	return nil
}

func (m MotionSubmitter) see(ctx context.Context, ds *datastore.Request, mperms *perm.MeetingPermission, motionSubmitterID int) (bool, error) {
	motionID, err := ds.MotionSubmitter_MotionID(motionSubmitterID).Value(ctx)
	if err != nil {
		return false, fmt.Errorf("getting motion id id: %w", err)
	}

	seeMotion, err := Motion{}.see(ctx, ds, mperms, motionID)
	if err != nil {
		return false, fmt.Errorf("checking motion %d can see: %w", motionID, err)
	}

	return seeMotion, nil
}
