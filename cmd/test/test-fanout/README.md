# Hybrid Fanout Test Script

This script automatically creates test data and verifies the hybrid fanout implementation.

## What It Does

1. **Creates 5 normal users**: alice, bob, charlie, diana, eve
2. **Creates 2 celebrity users**: celebrity1, celebrity2
3. **Sets up follow relationships**: All users follow each other + all celebrities
4. **Creates posts**: 2 posts per normal user, 3 posts per celebrity
5. **Tests timeline**: Fetches timeline and verifies streaming behavior

## How to Run

### Step 1: Ensure services are running

```bash
# Terminal 1
.\timeline-consumer.exe

# Terminal 2
.\server.exe
```

### Step 2: Build and run the test script

```bash
# Build the test script
go build -o test-fanout.exe ./cmd/test-fanout

# Run it
.\test-fanout.exe
```

### Step 3: Make celebrities "big personalities"

The script will output MongoDB commands. Run them in MongoDB shell:

```bash
mongosh

use microblogging

db.users.updateOne({_id: '<celebrity1_id>'}, {$set: {followerCount: 15000}})
db.users.updateOne({_id: '<celebrity2_id>'}, {$set: {followerCount: 15000}})
```

### Step 4: Run the test again

```bash
.\test-fanout.exe
```

Now you should see celebrity posts coming from the MongoDB chunk!

## Expected Output

```
=== Hybrid Fanout Test Script ===
Creating test data...

[1/5] Creating normal users...
  ✓ Created alice (ID: abc123...)
  ✓ Created bob (ID: def456...)
  ...

[2/5] Creating celebrity users...
  ✓ Created celebrity1 (ID: xyz789...)
  ℹ️  Run this in MongoDB to make celebrity1 a big personality:
     db.users.updateOne({_id: 'xyz789...'}, {$set: {followerCount: 15000}})

[3/5] Creating follow relationships...
  ✓ Created 30 follow relationships

[4/5] Creating posts...
  ✓ Created 10 posts from normal users
  ✓ Created 6 posts from celebrities

[5/5] Testing timeline retrieval...
Fetching timeline for alice...
  📦 Chunk 1 [REDIS]: 8 posts
  📦 Chunk 2 [MONGODB]: 6 posts [FINAL]

=== Test Results ===
Total posts in timeline: 14
Chunks received: 2

✓ Redis chunk received: 8 posts (from normal users)
✓ MongoDB chunk received: 6 posts (from big personalities)

=== Validation ===
✓ Received 2 chunks (Redis + MongoDB)
✓ Redis chunk present
✓ MongoDB chunk present
✓ Timeline has 14 posts

✅ Hybrid fanout test PASSED!
```

## What to Check

### Timeline Consumer Logs

Look for these messages when celebrities post:

```
Skipping fanout for big personality: xyz789...
```

If you DON'T see this message, the celebrity's followerCount is still < 10,000.

### Redis

Check that celebrity posts are NOT in user timelines:

```bash
redis-cli
> ZRANGE "timeline:alice_id" 0 -1
# Should NOT show celebrity post IDs (only normal user posts)
```

### MongoDB

Verify celebrity posts exist:

```javascript
db.posts.find({ authorId: "celebrity1_id" })
// Should return celebrity's posts
```

## Troubleshooting

**Issue: All posts in Redis chunk, MongoDB chunk empty**

→ Celebrities don't have followerCount >= 10,000. Run the MongoDB update commands.

**Issue: No posts in timeline**

→ Wait a few seconds after running the script, then run it again. RabbitMQ might be processing fanout.

**Issue: Script fails to connect**

→ Ensure server is running on the correct address (check config).
