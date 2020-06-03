BEGIN;
LOCK TABLE authorizations IN SHARE ROW EXCLUSIVE MODE;
INSERT INTO authorizations
    (slug, shared_key, public_key)
SELECT 'xxx-client-1', 'xxxx-yyyy-zzzz', 'somehash'
WHERE
    NOT EXISTS (
        SELECT slug FROM authorizations WHERE slug = 'xxx-client-1'
    );
COMMIT;
