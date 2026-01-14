# Feature: Go To Step for All Step Types

## Overview
Added "Go To Step" selector for all chatbot flow step types (previously only available for button steps via conditional_next).

## Changes Made

### Frontend (`frontend/src/views/chatbot/ChatbotFlowBuilderView.vue`)

**Added:** "Go To Step" selector after "Store Response As" field (lines ~1401-1420)

```vue
<!-- Go To Step (for all non-button steps) -->
<div v-if="selectedStep.message_type !== 'buttons'" class="space-y-1.5">
  <Label class="text-xs">Go To Step</Label>
  <Select v-model="selectedStep.next_step">
    <SelectTrigger class="h-8 text-xs">
      <SelectValue placeholder="Next step (sequential)" />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="">Next step (sequential)</SelectItem>
      <SelectItem
        v-for="step in stepsWithNames"
        :key="`goto-${step.step_name}`"
        :value="step.step_name"
      >
        {{ step.step_name }}
      </SelectItem>
    </SelectContent>
  </Select>
  <p class="text-xs text-muted-foreground">Jump to a specific step instead of the next one</p>
</div>
```

**Features:**
- ✅ Dropdown shows all steps with valid names
- ✅ Default option: "Next step (sequential)" (empty string = sequential flow)
- ✅ Excluded from button steps (they use conditional_next per button)
- ✅ Uses existing `next_step` field from the model

### Backend (No Changes Needed)

The backend already fully supports the `next_step` field:

**File:** `internal/handlers/chatbot_processor.go` (lines 1127-1130)

```go
// Determine next step
nextStepName := currentStep.NextStep
if nextStepName == "" && currentStepIndex+1 < len(flow.Steps) {
    nextStepName = flow.Steps[currentStepIndex+1].StepName
}
```

**Logic:**
1. Checks if `NextStep` is set on the step
2. If empty, defaults to next sequential step
3. For button steps, checks `ConditionalNext` for per-button routing

## Step Types Supported

This feature works for all step types:
- ✅ Text (`text`)
- ✅ Template (`template`)
- ✅ Script (`script`)
- ✅ API Fetch (`api_fetch`)
- ✅ Transfer (`transfer`)

Button steps (`buttons`) continue to use the per-button "Go to" via `conditional_next`.

## How It Works

### User Experience
1. User creates/edits a flow step (any type except buttons)
2. Sees "Go To Step" selector below "Store Response As"
3. Can choose:
   - **"Next step (sequential)"** - Default behavior, goes to next step in order
   - **Specific step name** - Jumps to that step, skipping others

### Backend Processing
1. When processing step completion, checks `step.NextStep`
2. If set, jumps to that step
3. If empty, proceeds to next sequential step
4. Same logic applies when skipping steps or processing no-input steps

## Testing

### Development Environment
Since hot reloading is enabled, changes are automatically reflected:

```bash
cd docker
docker compose -f docker-compose.dev.yml up -d
```

### Test Scenarios

1. **Sequential Flow (Default)**
   - Create flow: Step1 → Step2 → Step3
   - Leave "Go To Step" empty on all steps
   - Expected: Sequential execution

2. **Jump Forward**
   - Create flow: Step1 → Step2 → Step3
   - Set Step1 "Go To Step" = "Step3"
   - Expected: Step1 → Step3 (skips Step2)

3. **Conditional Logic**
   - Create flow with API step that sets a variable
   - Use "Skip Condition" on some steps
   - Use "Go To Step" to branch
   - Expected: Dynamic flow based on conditions

4. **Button Steps (Existing)**
   - Button steps continue using per-button "Go to"
   - Each button can route to different steps
   - Expected: No change to existing button behavior

## Database Schema

**Model:** `ChatbotFlowStep` (already has the field)

```go
type ChatbotFlowStep struct {
    // ...
    NextStep        string     `gorm:"size:100" json:"next_step"`
    ConditionalNext JSONB      `gorm:"type:jsonb" json:"conditional_next"` // For button steps
    // ...
}
```

No migration needed - field already exists in database.

## Benefits

1. **Flexibility:** Build non-linear flows without buttons
2. **Simplicity:** Single "Go To" field for most step types
3. **Consistency:** Same UX pattern as button steps
4. **Backward Compatible:** Empty value = sequential (existing behavior)

## Notes

- Button steps still use `conditional_next` for per-button routing
- Empty `next_step` maintains sequential flow (backward compatible)
- `stepsWithNames` computed property filters steps with valid names
- Hot reload enabled in dev environment for instant testing
