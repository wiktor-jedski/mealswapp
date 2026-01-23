# DisclaimerRenderer

**Traceability:** ARCH-015

## 1. Data Structures & Types

```typescript
interface DisclaimerContent {
  id: string;
  title: string;
  body: string;
  version: string;
  effectiveDate: string;
  lastUpdated: string;
}

interface DisclaimerConfig {
  displayOnLogin: boolean;
  displayOnAbout: boolean;
  showDismissButton: boolean;
  requireAcceptance: boolean;
  acceptedVersionKey: string;
}

interface DisclaimerState {
  isVisible: boolean;
  hasAccepted: boolean;
  acceptedVersion: string | null;
  isLoading: boolean;
  error: string | null;
}

interface DisclaimerRendererProps {
  location: 'login' | 'about';
  config?: Partial<DisclaimerConfig>;
}
```

## 2. Logic & Algorithms

### 2.1 Component Initialization Flow

1. Initialize component with props (location: 'login' | 'about')
2. Load disclaimer configuration from store/config
3. Check localStorage for previously accepted disclaimer version
4. Fetch latest disclaimer content from API
5. Compare accepted version with current version
6. Determine visibility state based on:
   - Has user accepted current version?
   - Is disclaimer required for this location?
   - Has user dismissed disclaimer previously?

### 2.2 Disclaimer Display Logic

```
FUNCTION shouldShowDisclaimer(location, userState, config):
  IF config.requireAcceptance AND location === 'login':
    RETURN NOT userState.hasAccepted OR userState.acceptedVersion !== currentVersion

  IF config.displayOnAbout AND location === 'about':
    RETURN NOT userState.hasAccepted OR userState.acceptedVersion !== currentVersion

  IF config.showDismissButton AND userState.isDismissed:
    RETURN false

  RETURN true
END FUNCTION
```

### 2.3 Acceptance Handler

```
FUNCTION handleAcceptance():
  1. Set loading state to true
  2. Call API endpoint POST /api/compliance/disclaimer/accept
     Payload: { version: currentDisclaimer.version, location: props.location }
  3. On success:
     - Store accepted version in localStorage
     - Update Svelte store with new acceptance state
     - Hide disclaimer component
  4. On error:
     - Set error state with error message
     - Log error to monitoring service
END FUNCTION
```

### 2.4 Dismiss Handler

```
FUNCTION handleDismiss():
  1. Store dismissal timestamp in localStorage
     Key: dismissal_${location}_${currentDate}
  2. Update local state to hide disclaimer
  3. If dismiss button is not allowed per config, this handler is not exposed
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 State Transitions

| Current State | Event | Next State |
|---------------|-------|------------|
| initial | loadComplete | visible / hidden |
| visible | acceptClicked | accepting |
| accepting | API success | hidden |
| accepting | API error | error |
| error | retryClicked | accepting |
| hidden | versionChanged | visible |
| hidden | dismissClicked | dismissed |

### 3.2 Error States

| Error Type | Trigger | User Feedback | Recovery Action |
|------------|---------|---------------|-----------------|
| FetchError | API /api/disclaimer returns non-200 | "Failed to load disclaimer. Please refresh." | Retry button, auto-retry after 30s |
| NetworkError | Network timeout/offline | "Connection lost. Checking again..." | Auto-retry when connectivity restored |
| AcceptanceError | POST /accept returns error | "Unable to save acceptance. Please try again." | Retry button, logout if persistent |
| VersionMismatch | Accepted version !== current | Show updated disclaimer | Require new acceptance |

### 3.3 Svelte Store Structure

```typescript
// stores/disclaimer.ts
interface DisclaimerStore {
  currentContent: DisclaimerContent | null;
  userAcceptance: {
    version: string | null;
    acceptedAt: string | null;
    locations: string[];
  };
  config: DisclaimerConfig;
}

export const disclaimerStore = writable<DisclaimerStore>({
  currentContent: null,
  userAcceptance: { version: null, acceptedAt: null, locations: [] },
  config: { displayOnLogin: true, displayOnAbout: true, showDismissButton: true, requireAcceptance: true, acceptedVersionKey: 'disclaimer_accepted_version' }
});
```

## 4. Component Interfaces

### 4.1 Main Component Signature

```svelte
<script lang="ts">
  import type { DisclaimerRendererProps } from './types';

  export let location: DisclaimerRendererProps['location'];
  export let config: DisclaimerRendererProps['config'] = {};

  // Reactive derived state
  $: effectiveConfig = { ...defaultConfig, ...config };
  $: shouldShow = $disclaimerStore.currentContent !== null && 
                  shouldShowDisclaimer(location, $disclaimerStore.userAcceptance, effectiveConfig);
</script>
```

### 4.2 Internal Functions

```typescript
function loadDisclaimerContent(): Promise<void>
  Input: none
  Output: Promise resolving to DisclaimerContent or rejecting with FetchError
  Side Effects: Updates $disclaimerStore.currentContent

function acceptDisclaimer(): Promise<void>
  Input: none
  Output: Promise resolving on success, rejecting with AcceptanceError on failure
  Side Effects: Updates localStorage, calls API, updates $disclaimerStore.userAcceptance

function dismissDisclaimer(): void
  Input: none
  Output: void
  Side Effects: Updates localStorage with dismissal timestamp

function retryLoad(): void
  Input: none
  Output: void
  Side Effects: Calls loadDisclaimerContent()

function getStoredAcceptance(): { version: string | null; timestamp: string | null }
  Input: effectiveConfig.acceptedVersionKey
  Output: Object with version and timestamp from localStorage

function checkVersionUpdate(): boolean
  Input: storedVersion, currentContent.version
  Output: true if versions differ, false otherwise
```

### 4.3 API Endpoints

```
GET /api/compliance/disclaimer
  Response: {
    success: true,
    data: DisclaimerContent
  }
  Errors: 404 (No disclaimer configured), 500 (Server error)

POST /api/compliance/disclaimer/accept
  Body: { version: string, location: string }
  Response: {
    success: true,
    data: { acceptedAt: timestamp }
  }
  Errors: 400 (Invalid version), 409 (Version mismatch), 500 (Server error)
```

### 4.4 Tailwind CSS Classes

```svelte
<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
  <div class="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 overflow-hidden">
    <div class="p-6">
      <h2 class="text-xl font-bold mb-4">{content.title}</h2>
      <div class="prose prose-sm max-h-96 overflow-y-auto">
        {@html content.body}
      </div>
    </div>
    <div class="bg-gray-50 px-6 py-4 flex justify-between">
      {#if config.showDismissButton}
        <button class="text-gray-600 hover:text-gray-800" on:click={dismissDisclaimer}>
          Dismiss
        </button>
      {/if}
      <button 
        class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        on:click={acceptDisclaimer}
        disabled={isAccepting}
      >
        {isAccepting ? 'Saving...' : 'I Accept'}
      </button>
    </div>
  </div>
</div>
```
