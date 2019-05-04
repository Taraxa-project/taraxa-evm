package util

type revision [2]int
type Revert func()

type RevertLog struct {
	revisions []revision
	reverts   []Revert
}

func (this *RevertLog) Append(revert Revert) {
	this.reverts = append(this.reverts, revert)
}

func (this *RevertLog) CurrentSnapshot() int {
	if revCount := len(this.revisions); revCount > 0 {
		return this.revisions[revCount-1][0]
	}
	return -1
}

func (this *RevertLog) Snapshot(snapshotId int) {
	Assert(this.CurrentSnapshot() < snapshotId)
	this.revisions = append(this.revisions, revision{snapshotId, len(this.reverts)})
}

func (this *RevertLog) RevertToSnapshot(snapshotId int) {
	var revisionsIndex, revertsIndex int
	for i := len(this.revisions) - 1; i >= 0; i-- {
		revision := this.revisions[i]
		if revision[0] == snapshotId {
			revisionsIndex, revertsIndex = i, revision[1]
			break
		}
	}
	for i := len(this.reverts) - 1; i >= revertsIndex; i-- {
		this.reverts[i]()
	}
	this.revisions, this.reverts = this.revisions[:revisionsIndex], this.reverts[:revertsIndex]
}
