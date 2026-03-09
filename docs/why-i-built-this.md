# Why I Built This

I didn't set out to build Kiloforge. I set out to write code faster, and Kiloforge is what happened when I stopped being able to keep up with my own tools.

## How It Started

My first encounter with an agentic coding tool was Augment Code, around mid-2025. I'd seen Copilot and other autocomplete tools before and honestly wasn't impressed — they felt like they created more work than they saved, constantly reworking suggestions that weren't quite right. Augment was different, but only once a researcher I worked with told me to turn on "auto" mode. That was the moment everything changed.

I was blown away. When directed well, the agent was producing real, working code at a pace I couldn't match manually. From that day forward, I wrote very little code myself. Even for small things I could have done in minutes, I found myself describing the task to an agent instead. I started telling everyone I knew that I feared for developer jobs within the next six months to a year.

I moved to Antigravity in December 2025. Excellent model, but the interface had real problems — model selection was global across all windows, working with multiple copies of a repo was painful, and even with a Google AI Pro subscription I was blowing through the quota every five hours. They had a hidden weekly quota too. I ended up buying two additional Pro subscriptions just to keep working.

## The Week Everything Changed

I first tried Claude Code on Friday, February 27th, 2026. Just tested it out — seemed solid. I started migrating my skills over, including a set I'd adapted from "Gemini Conductor" — a track-based development workflow I'd been refining to work better with agentic tools. At the same time, I decided to experiment with git worktrees for parallel agent work.

That Sunday, I pushed as hard as I could. I started generating tracks and passing them to worker branches in separate worktrees. The hit rate was incredibly high — agents were producing working code on the first pass. But I immediately ran into a new problem: managing which worker was doing what. I had agents pausing before merge, and I was manually telling each one when to proceed. If I didn't, they'd all try to merge at once and create conflicts everywhere.

**I had become a human mutex.**

The realization hit me and the solution was obvious: update the skills to implement an actual mutex protocol. Agents would finish their work, acquire the lock, run a strict verification process — build, lint, typecheck, tests — rebase onto main, fix any conflicts, verify again, merge, and release the lock. All autonomously.

## The Numbers

The results were immediate and staggering.

Before worktrees, with one or two agents and a basic workflow, I was getting 30–40 commits per day and completing about 3 tracks on average.

On March 2nd, with 4 agents and manual management over just a few hours, I hit 160 commits in a single day.

On March 5th, with 8 workers and the automated mutex, I peaked at 240 commits in a single day and was completing around 30 tracks.

That's not an incremental improvement. That's a phase change. I was getting literally hundreds of times the development speed I'd had only weeks earlier. I was so shocked that I started telling everyone — loved ones, family, friends, anyone who would listen. Most of them aren't developers, so it was hard to convey why this was such an incredibly massive deal.

And then something unexpected happened: I started running out of ideas faster than agents could finish work. So I created research tracks — agents investigating similar products, analyzing gaps, suggesting new features. Then I had agents review the research and generate track proposals, prioritized by common gaps across the research. I'd curate the suggestions, keep the ones aligned with my vision, prune the rest, and push them to workers. The pipeline fed itself.

## The Philosophy

Building this way taught me something I didn't expect: **you don't need to review agent code if the process is solid.** If you do, you become the bottleneck.

My original plan was to have agents collaborate through Gitea PRs — automated review cycles where a human could jump in and participate. Through building Kiloforge, I came to believe that reviewing agent code, when they have a well-designed working process, just slows things down. If we can design the process so that we have high confidence agents will implement what we want, why review?

Even in traditional development, there's drift from standards. Minor shifts create "legacy" implementations over time. That's normal. When we detect sufficient misalignment, we do an audit and provide better guidance going forward. The same applies here. Why agonize over every line when mass realignment is trivial? Just push an audit track to a worker.

## Why "Kiloforge"

The name is quite literal. "Kilo" — a thousand. I believe my development speed with this process is genuinely around 1,000 times what it was before I started using agentic tools. And the thing is, what I was seeing those numbers with — that was still me doing things manually, without Kiloforge itself. Kiloforge is designed to make this process seamless, smooth, and to give you the information you need to keep improving it: thorough metrics, creation-to-completion tracing, complete history, and the ability to revive any agent that worked on something and ask it questions.

## The Proof

If this sounds too good to be true, just look at the project itself. I initialized this repository three days ago. The entire development history is here — every track, every commit, every architectural decision. Over a hundred tracks implemented, verified, and merged. The project is its own testament to the process that built it.

I don't think this is just awesome or amazing. I think it is literally world-changing.

---

*— Ben Baldivia, 2026 March 9*
