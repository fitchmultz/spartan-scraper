/**
 * Job List Component
 *
 * Displays the list of active/completed jobs with their status, kind, timestamps,
 * and error messages. Provides action buttons for viewing results, canceling running
 * jobs, and deleting completed jobs. Supports refresh to update job states.
 *
 * @module JobList
 */
import { useState, useEffect } from "react";
import { statusClass } from "../lib/form-utils";
import type { JobEntry } from "../types";

interface JobListProps {
  jobs: JobEntry[];
  error: string | null;
  onViewResults: (jobId: string, format: string, page: number) => void;
  onCancel: (jobId: string) => void;
  onDelete: (jobId: string) => void;
  onRefresh: () => void;
  currentPage: number;
  totalJobs: number;
  jobsPerPage: number;
  onPageChange: (page: number) => void;
}

export function JobList({
  jobs,
  error,
  onViewResults,
  onCancel,
  onDelete,
  onRefresh,
  currentPage,
  totalJobs,
  jobsPerPage,
  onPageChange,
}: JobListProps) {
  const [jumpInputValue, setJumpInputValue] = useState(currentPage.toString());

  useEffect(() => {
    setJumpInputValue(currentPage.toString());
  }, [currentPage]);

  const maxPage = Math.ceil(totalJobs / jobsPerPage);

  return (
    <section className="panel">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <h2>Active Jobs</h2>
        <button type="button" className="secondary" onClick={onRefresh}>
          Refresh
        </button>
      </div>
      {error ? <p className="error">{error}</p> : null}

      {totalJobs > jobsPerPage ? (
        <div className="pagination-controls" style={{ marginTop: 12 }}>
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => onPageChange(currentPage - 1)}
          >
            Previous
          </button>

          <span className="pagination-info">
            Page {currentPage} of {maxPage} ({totalJobs} total jobs)
          </span>

          <button
            type="button"
            disabled={currentPage >= maxPage}
            onClick={() => onPageChange(currentPage + 1)}
          >
            Next
          </button>

          <div className="pagination-jump">
            <input
              type="number"
              min="1"
              max={maxPage}
              value={jumpInputValue}
              onChange={(e) => {
                const page = parseInt(e.target.value, 10);
                if (Number.isInteger(page) && page >= 1 && page <= maxPage) {
                  setJumpInputValue(e.target.value);
                }
              }}
            />
            <button
              type="button"
              onClick={() => {
                const page = parseInt(jumpInputValue, 10);
                if (page >= 1 && page <= maxPage) {
                  onPageChange(page);
                }
              }}
            >
              Go
            </button>
          </div>
        </div>
      ) : null}

      <div className="job-list" style={{ marginTop: 12 }}>
        {jobs.length === 0 ? (
          <div>No jobs yet. Submit a scrape or crawl.</div>
        ) : (
          jobs.map((job) => (
            <div key={job.id} className="job-item">
              <div>{job.id}</div>
              <div>
                <span className={`badge ${statusClass(job.status ?? "")}`}>
                  {job.status}
                </span>{" "}
                {job.kind}
              </div>
              <div>Updated: {job.updatedAt}</div>
              {job.error ? <div>Error: {job.error}</div> : null}
              <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
                {job.status === "succeeded" ? (
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => onViewResults(job.id ?? "", "jsonl", 1)}
                  >
                    View Results
                  </button>
                ) : null}
                {job.status === "queued" || job.status === "running" ? (
                  <button
                    type="button"
                    className="secondary"
                    onClick={() => onCancel(job.id ?? "")}
                  >
                    Cancel
                  </button>
                ) : null}
                <button
                  type="button"
                  className="secondary"
                  onClick={() => onDelete(job.id ?? "")}
                  style={{ color: "#ff6b6b" }}
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
