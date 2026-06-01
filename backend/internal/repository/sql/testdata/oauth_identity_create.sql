-- Implements DESIGN-006 OAuthHandler integration fixture.
INSERT INTO oauth_identities (user_id, provider, provider_user_id, email)
VALUES ($1, 'google', 'google-user-1', 'oauth-only@example.test');
