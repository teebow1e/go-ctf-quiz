#!/usr/bin/env python3
"""
CTF Quiz Log Analyzer with Visualizations
Analyzes quiz_attempts.json log file to generate insights and visualizations
"""

import json
import sys
from collections import defaultdict, Counter
from pathlib import Path

try:
    import matplotlib.pyplot as plt
    import matplotlib
    matplotlib.use('Agg')  # Non-interactive backend
    HAS_MATPLOTLIB = True
except ImportError:
    print("Warning: matplotlib not installed. Install with: pip install matplotlib")
    HAS_MATPLOTLIB = False


def load_logs(log_file="quiz_attempts.json"):
    """Load NDJSON log file"""
    logs = []
    try:
        with open(log_file, 'r') as f:
            for line in f:
                if line.strip():
                    logs.append(json.loads(line))
        return logs
    except FileNotFoundError:
        print(f"Error: {log_file} not found!")
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON: {e}")
        sys.exit(1)


def load_questions(question_file="question.json"):
    """Load question file to get correct answers"""
    try:
        with open(question_file, 'r') as f:
            quiz_data = json.load(f)
        # Create lookup dict: question_id -> correct_answer
        return {q['id']: q['answer'] for q in quiz_data['questions']}
    except FileNotFoundError:
        print(f"Warning: {question_file} not found! Cannot determine correct answers.")
        return {}


def analyze_question_difficulty(logs, correct_answers):
    """Analyze which questions are most difficult"""
    question_stats = defaultdict(lambda: {'total': 0, 'correct': 0, 'incorrect': 0})

    for log in logs:
        for q in log['quiz_attempt']:
            qid = q['question_id']
            question_stats[qid]['total'] += 1
            if q['status'] == 'correct':
                question_stats[qid]['correct'] += 1
            else:
                question_stats[qid]['incorrect'] += 1

    # Calculate success rate
    results = []
    for qid, stats in sorted(question_stats.items()):
        success_rate = (stats['correct'] / stats['total'] * 100) if stats['total'] > 0 else 0
        results.append({
            'question_id': qid,
            'total_attempts': stats['total'],
            'correct': stats['correct'],
            'incorrect': stats['incorrect'],
            'success_rate': success_rate
        })

    return sorted(results, key=lambda x: x['success_rate'])


def analyze_wrong_answers(logs, correct_answers):
    """Analyze common wrong answers for each question"""
    wrong_answers = defaultdict(lambda: Counter())

    for log in logs:
        for q in log['quiz_attempt']:
            if q['status'] == 'incorrect':
                qid = q['question_id']
                wrong_answers[qid][q['answer']] += 1

    return wrong_answers


def analyze_failure_points(logs):
    """Analyze where users typically fail"""
    failure_points = Counter()

    for log in logs:
        if log['status'] in ['wrong_answer', 'time_out']:
            # The question they failed on is the last one in quiz_attempt
            if log['quiz_attempt']:
                last_question = log['quiz_attempt'][-1]
                failure_points[last_question['question_id']] += 1

    return failure_points


def analyze_retry_patterns(logs):
    """Analyze retry behavior"""
    user_attempts = defaultdict(list)

    for log in logs:
        user_attempts[log['user_token']].append({
            'retry_count': log['retry_count'],
            'status': log['status'],
            'questions_answered': log['questions_answered']
        })

    # Stats
    total_users = len(user_attempts)
    users_who_retry = sum(1 for attempts in user_attempts.values() if len(attempts) > 1)
    max_retries = max((len(attempts) for attempts in user_attempts.values()), default=0)

    # Retry count distribution
    retry_distribution = Counter(log['retry_count'] for log in logs)

    return {
        'total_users': total_users,
        'users_who_retry': users_who_retry,
        'retry_percentage': (users_who_retry / total_users * 100) if total_users > 0 else 0,
        'max_retries': max_retries,
        'retry_distribution': retry_distribution
    }


def analyze_completion_stats(logs):
    """Analyze overall completion statistics"""
    total_attempts = len(logs)
    status_counts = Counter(log['status'] for log in logs)

    return {
        'total_attempts': total_attempts,
        'completed': status_counts['completed'],
        'failed_wrong_answer': status_counts['wrong_answer'],
        'failed_timeout': status_counts['time_out'],
        'failed_error': status_counts['server_runtime_error'],
        'completion_rate': (status_counts['completed'] / total_attempts * 100) if total_attempts > 0 else 0
    }


def visualize_question_difficulty(difficulty_data, output_file='question_difficulty.png'):
    """Create bar chart of question success rates"""
    if not HAS_MATPLOTLIB:
        return

    fig, ax = plt.subplots(figsize=(12, 6))

    question_ids = [str(q['question_id']) for q in difficulty_data]
    success_rates = [q['success_rate'] for q in difficulty_data]

    bars = ax.bar(question_ids, success_rates, color=['red' if sr < 50 else 'orange' if sr < 75 else 'green' for sr in success_rates])

    ax.set_xlabel('Question ID', fontsize=12)
    ax.set_ylabel('Success Rate (%)', fontsize=12)
    ax.set_title('Question Difficulty - Success Rate by Question', fontsize=14, fontweight='bold')
    ax.set_ylim(0, 100)
    ax.axhline(y=50, color='gray', linestyle='--', alpha=0.5, label='50% threshold')
    ax.legend()

    plt.tight_layout()
    plt.savefig(output_file, dpi=300)
    print(f"✅ Saved visualization: {output_file}")
    plt.close()


def visualize_completion_stats(completion_data, output_file='completion_stats.png'):
    """Create pie chart of completion statistics"""
    if not HAS_MATPLOTLIB:
        return

    fig, ax = plt.subplots(figsize=(10, 8))

    labels = []
    sizes = []
    colors = []

    if completion_data['completed'] > 0:
        labels.append(f"Completed ({completion_data['completed']})")
        sizes.append(completion_data['completed'])
        colors.append('green')

    if completion_data['failed_wrong_answer'] > 0:
        labels.append(f"Failed - Wrong Answer ({completion_data['failed_wrong_answer']})")
        sizes.append(completion_data['failed_wrong_answer'])
        colors.append('red')

    if completion_data['failed_timeout'] > 0:
        labels.append(f"Failed - Timeout ({completion_data['failed_timeout']})")
        sizes.append(completion_data['failed_timeout'])
        colors.append('orange')

    if completion_data['failed_error'] > 0:
        labels.append(f"Failed - Error ({completion_data['failed_error']})")
        sizes.append(completion_data['failed_error'])
        colors.append('gray')

    ax.pie(sizes, labels=labels, colors=colors, autopct='%1.1f%%', startangle=90)
    ax.set_title(f'Quiz Completion Statistics\nTotal Attempts: {completion_data["total_attempts"]}',
                 fontsize=14, fontweight='bold')

    plt.tight_layout()
    plt.savefig(output_file, dpi=300)
    print(f"✅ Saved visualization: {output_file}")
    plt.close()


def visualize_failure_points(failure_data, output_file='failure_points.png'):
    """Create bar chart showing where users fail most"""
    if not HAS_MATPLOTLIB or not failure_data:
        return

    fig, ax = plt.subplots(figsize=(12, 6))

    questions = [str(qid) for qid, _ in failure_data.most_common()]
    failures = [count for _, count in failure_data.most_common()]

    ax.bar(questions, failures, color='red', alpha=0.7)

    ax.set_xlabel('Question ID', fontsize=12)
    ax.set_ylabel('Number of Failures', fontsize=12)
    ax.set_title('Failure Points - Where Users Give Up', fontsize=14, fontweight='bold')

    plt.tight_layout()
    plt.savefig(output_file, dpi=300)
    print(f"✅ Saved visualization: {output_file}")
    plt.close()


def visualize_wrong_answers(wrong_answers, question_id, correct_answer, output_file=None):
    """Create bar chart of wrong answers for a specific question"""
    if not HAS_MATPLOTLIB:
        return

    if question_id not in wrong_answers or not wrong_answers[question_id]:
        return

    if output_file is None:
        output_file = f'wrong_answers_q{question_id}.png'

    fig, ax = plt.subplots(figsize=(12, 6))

    top_wrong = wrong_answers[question_id].most_common(10)
    answers = [ans[:30] + '...' if len(ans) > 30 else ans for ans, _ in top_wrong]  # Truncate long answers
    counts = [count for _, count in top_wrong]

    ax.barh(answers, counts, color='crimson', alpha=0.7)

    ax.set_xlabel('Count', fontsize=12)
    ax.set_ylabel('Wrong Answer', fontsize=12)
    ax.set_title(f'Common Wrong Answers - Question {question_id}\nCorrect Answer: {correct_answer}',
                 fontsize=14, fontweight='bold')
    ax.invert_yaxis()

    plt.tight_layout()
    plt.savefig(output_file, dpi=300)
    print(f"✅ Saved visualization: {output_file}")
    plt.close()


def visualize_retry_distribution(retry_data, output_file='retry_distribution.png'):
    """Create bar chart of retry attempt distribution"""
    if not HAS_MATPLOTLIB:
        return

    fig, ax = plt.subplots(figsize=(10, 6))

    retry_counts = sorted(retry_data['retry_distribution'].keys())
    frequencies = [retry_data['retry_distribution'][rc] for rc in retry_counts]

    ax.bar(retry_counts, frequencies, color='steelblue', alpha=0.7)

    ax.set_xlabel('Retry Attempt Number', fontsize=12)
    ax.set_ylabel('Number of Attempts', fontsize=12)
    ax.set_title('Retry Attempt Distribution', fontsize=14, fontweight='bold')

    plt.tight_layout()
    plt.savefig(output_file, dpi=300)
    print(f"✅ Saved visualization: {output_file}")
    plt.close()


def print_report(logs, correct_answers):
    """Print comprehensive analysis report"""
    print("=" * 80)
    print("CTF QUIZ LOG ANALYSIS REPORT")
    print("=" * 80)
    print()

    # Overall stats
    print("📊 OVERALL STATISTICS")
    print("-" * 80)
    completion = analyze_completion_stats(logs)
    print(f"Total Attempts:        {completion['total_attempts']}")
    print(f"Completed:             {completion['completed']} ({completion['completion_rate']:.1f}%)")
    print(f"Failed (Wrong Answer): {completion['failed_wrong_answer']}")
    print(f"Failed (Timeout):      {completion['failed_timeout']}")
    print(f"Failed (Error):        {completion['failed_error']}")
    print()

    # Retry patterns
    print("🔄 RETRY PATTERNS")
    print("-" * 80)
    retry_stats = analyze_retry_patterns(logs)
    print(f"Total Users:           {retry_stats['total_users']}")
    print(f"Users Who Retry:       {retry_stats['users_who_retry']} ({retry_stats['retry_percentage']:.1f}%)")
    print(f"Max Retries by User:   {retry_stats['max_retries']}")
    print()

    # Question difficulty
    print("📈 QUESTION DIFFICULTY (Hardest to Easiest)")
    print("-" * 80)
    print(f"{'Q ID':<6} {'Total':<8} {'Correct':<10} {'Incorrect':<12} {'Success Rate':<12}")
    print("-" * 80)
    difficulty = analyze_question_difficulty(logs, correct_answers)
    for q in difficulty:  # Already sorted by difficulty
        print(f"{q['question_id']:<6} {q['total_attempts']:<8} {q['correct']:<10} {q['incorrect']:<12} {q['success_rate']:<11.1f}%")
    print()

    # Failure points
    print("❌ FAILURE POINTS (Where users give up)")
    print("-" * 80)
    failure_points = analyze_failure_points(logs)
    if failure_points:
        print(f"{'Question ID':<15} {'Failures':<10}")
        print("-" * 80)
        for qid, count in failure_points.most_common():
            print(f"{qid:<15} {count:<10}")
    else:
        print("No failure data available.")
    print()

    # Common wrong answers
    print("🚫 COMMON WRONG ANSWERS")
    print("-" * 80)
    wrong_answers = analyze_wrong_answers(logs, correct_answers)
    for qid in sorted(wrong_answers.keys()):
        print(f"\nQuestion {qid}:")
        if qid in correct_answers:
            print(f"  Correct Answer: {correct_answers[qid]}")
        print("  Common Wrong Answers:")
        for answer, count in wrong_answers[qid].most_common(5):
            print(f"    - '{answer}': {count} times")
    print()

    print("=" * 80)


def export_csv(logs, output_file="quiz_analysis.csv"):
    """Export flattened log data to CSV for external tools"""
    import csv

    with open(output_file, 'w', newline='') as f:
        writer = csv.writer(f)
        # Header
        writer.writerow([
            'username', 'user_token', 'timestamp', 'status', 'retry_count',
            'questions_answered', 'question_id', 'question_text',
            'user_answer', 'is_correct'
        ])

        # Data
        for log in logs:
            for q in log['quiz_attempt']:
                writer.writerow([
                    log['username'],
                    log['user_token'],
                    log['timestamp'],
                    log['status'],
                    log['retry_count'],
                    log['questions_answered'],
                    q['question_id'],
                    q['question'],
                    q['answer'],
                    q['status']
                ])

    print(f"✅ Exported to {output_file}")


def main():
    if len(sys.argv) > 1:
        log_file = sys.argv[1]
    else:
        log_file = "quiz_attempts.json"

    print(f"Loading logs from {log_file}...")
    logs = load_logs(log_file)

    if len(logs) == 0:
        print("No logs found!")
        return

    print(f"Loaded {len(logs)} log entries.\n")

    # Try to load correct answers
    correct_answers = load_questions()

    # Generate report
    print_report(logs, correct_answers)

    # Generate visualizations
    if HAS_MATPLOTLIB:
        print("\n" + "=" * 80)
        print("GENERATING VISUALIZATIONS")
        print("=" * 80 + "\n")

        completion = analyze_completion_stats(logs)
        difficulty = analyze_question_difficulty(logs, correct_answers)
        failure_points = analyze_failure_points(logs)
        wrong_answers = analyze_wrong_answers(logs, correct_answers)
        retry_stats = analyze_retry_patterns(logs)

        visualize_completion_stats(completion)
        visualize_question_difficulty(difficulty)
        visualize_failure_points(failure_points)
        visualize_retry_distribution(retry_stats)

        # Generate wrong answer charts for top 3 hardest questions
        for q in difficulty[:3]:
            qid = q['question_id']
            if qid in wrong_answers and wrong_answers[qid]:
                correct = correct_answers.get(qid, "Unknown")
                visualize_wrong_answers(wrong_answers, qid, correct)

        print("\n✅ All visualizations generated!")
    else:
        print("\n⚠️  matplotlib not installed. Skipping visualizations.")
        print("Install with: pip install matplotlib")

    # Optional: Export to CSV
    print("\n" + "=" * 80)
    response = input("Export raw data to CSV? (y/n): ")
    if response.lower() == 'y':
        export_csv(logs)


if __name__ == "__main__":
    main()
