package vision

import "testing"

func TestAppendKeyFrameToSceneMutatesOriginalScene(t *testing.T) {
	scenes := []SceneInfo{
		{ID: 7, Start: 0, End: 5, Duration: 5},
		{ID: 8, Start: 5, End: 10, Duration: 5},
	}

	first := KeyFrame{Index: 0, SceneID: 7, Timestamp: 2.5}
	second := KeyFrame{Index: 1, SceneID: 8, Timestamp: 7.5}

	appendKeyFrameToScene(scenes, 0, first)
	appendKeyFrameToScene(scenes, 1, second)

	if len(scenes[0].KeyFrames) != 1 {
		t.Fatalf("expected first scene to keep 1 key frame, got %d", len(scenes[0].KeyFrames))
	}
	if len(scenes[1].KeyFrames) != 1 {
		t.Fatalf("expected second scene to keep 1 key frame, got %d", len(scenes[1].KeyFrames))
	}
	if scenes[0].KeyFrames[0].SceneID != 7 {
		t.Fatalf("expected first scene key frame scene id 7, got %d", scenes[0].KeyFrames[0].SceneID)
	}
	if scenes[1].KeyFrames[0].SceneID != 8 {
		t.Fatalf("expected second scene key frame scene id 8, got %d", scenes[1].KeyFrames[0].SceneID)
	}
}

func TestAppendKeyFrameToSceneIgnoresOutOfRangeIndex(t *testing.T) {
	scenes := []SceneInfo{{ID: 1}}
	appendKeyFrameToScene(scenes, -1, KeyFrame{SceneID: 1})
	appendKeyFrameToScene(scenes, 1, KeyFrame{SceneID: 1})

	if len(scenes[0].KeyFrames) != 0 {
		t.Fatalf("expected no key frames to be appended for invalid indices, got %d", len(scenes[0].KeyFrames))
	}
}
