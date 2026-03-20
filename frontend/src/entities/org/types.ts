export type Person = {
  id: number;
  fullName: string;
  role: string;
  velocity: number;
  isActive: boolean;
  teamId: number;
  teamLeadId?: number;
};

export type Team = {
  id: number;
  name: string;
  leadId?: number;
  members: Person[];
};

export type OrgStructureResponse = {
  updatedAt: string;
  teams: Team[];
};

export type PeopleStats = {
  totalPeople: number;
  activePeople: number;
  averageVelocity: number;
};
